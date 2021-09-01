// Copyright 2020 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		redirect(w, "/metrics.html")
		return
	}
	if path == "/plot/remove" {
		err = handleRemovePlot(req.Form)
		if err != nil {
			errorResponse(w, err)
			return
		}
		redirect(w, "/metrics.html")
		return
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
