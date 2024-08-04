package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-playground/form/v4"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "url", uri)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data templateData) {
	// TODO: remove this template cache before deployment
	var err error
	app.templateCache, err = newTemplateCache()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// retrieve template set from cache based on page name
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, r, err)
		return
	}

	// initialize new buffer
	buf := new(bytes.Buffer)

	// Write template to buffer instead of straight to the
	// http.ResponseWriter. If there is an error, call serverError()
	// helper and then return
	fmt.Printf("Template data for %s: %+v\n", page, data)
	ts.ExecuteTemplate(os.Stdout, "base", data)
	// TODO: be sure to add the : after removing the above code
	err = ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// TODO: remove these headers when done testing
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// write out status code
	w.WriteHeader(status)

	fmt.Printf("Rendering %s template. IsAuthenticated: %v\n", page, data.IsAuthenticated)
	// write contents of buffer to http.ResponseWriter
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}
		return err
	}
	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	fmt.Printf("isAuthenticated called: id = %v\n", id)
	return id > 0
}
