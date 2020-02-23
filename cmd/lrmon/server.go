package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/erkkah/letarette/pkg/logger"
)

type server struct {
	http.Server
	lookupTemplate func(path string) *template.Template
}

func (s *server) run(addr string) {
	s.Addr = addr
	s.Handler = s
	logger.Info.Printf("Listening on %v", addr)
	log.Fatal(s.ListenAndServe())
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI
	parsed, err := url.Parse(uri)
	if err != nil {
		errorResponse(w, err)
		return
	}
	err = req.ParseForm()
	if err != nil {
		errorResponse(w, err)
		return
	}
	logger.Debug.Printf("%s", parsed.Path)
	path := parsed.Path
	if path == "/plot/add" {
		err = handleAddPlot(req.Form)
		if err != nil {
			errorResponse(w, err)
			return
		}
		path = "/metrics.html"
	}
	if path == "/plot/remove" {
		err = handleRemovePlot(req.Form)
		if err != nil {
			errorResponse(w, err)
			return
		}
		path = "/metrics.html"
	}
	if path == "/" {
		path = "/index.html"
	}
	if template := s.lookupTemplate(path); template != nil {
		state := getState()
		ctx := Context{
			State:   state,
			Request: req,
		}
		setContentTypeFromPath(w, path)
		err = template.Execute(w, ctx)
	} else {
		err = textResponse(w, 404, "Not found")
	}
	if err != nil {
		logger.Error.Printf("Error: %v", err)
	}
}

// Context is the template rendering context
type Context struct {
	State   State
	Request *http.Request
}
