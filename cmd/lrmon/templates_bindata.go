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

// +build prod

//go:generate go run github.com/go-bindata/go-bindata/go-bindata -pkg $GOPACKAGE -prefix static/ -o bindata.go static/...

package main

import (
	"html/template"
	"log"

	"github.com/erkkah/letarette/pkg/logger"
)

var loadedTemplates *template.Template

func lookupTemplate(path string) *template.Template {
	return loadedTemplates.Lookup(path)
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
	for _, file := range AssetNames() {
		logger.Debug.Printf("Parsing template %q", file)
		templateData, err := Asset(file)
		if err != nil {
			return nil, err
		}
		templateName := "/" + file
		_, err = result.New(templateName).Parse(string(templateData))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
