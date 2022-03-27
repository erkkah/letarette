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

//go:build prod
// +build prod

package main

import (
	"embed"
	"html/template"
	"io"
	"log"
	"strings"

	"github.com/erkkah/letarette/pkg/logger"
)

//go:embed static
var templateFS embed.FS
var loadedTemplates *template.Template

func lookupTemplate(path string) *template.Template {
	return loadedTemplates.Lookup(path)
}

func serveRaw(path string, writer io.Writer) error {
	file, err := templateFS.Open("static/" + path)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}

func init() {
	var err error
	loadedTemplates, err = parseTemplates()
	if err != nil {
		log.Panicf("Template parse error: %v", err)
	}
}

func parseTemplates() (*template.Template, error) {
	result := template.New("").Funcs(templateFunctions)
	entries, err := templateFS.ReadDir("static")
	if err != nil {
		return nil, err
	}

	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}
		logger.Debug.Printf("Parsing template %q", file)
		templateData, err := templateFS.ReadFile("static/" + file.Name())
		if err != nil {
			return nil, err
		}
		templateName := "/" + file.Name()
		_, err = result.New(templateName).Parse(string(templateData))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
