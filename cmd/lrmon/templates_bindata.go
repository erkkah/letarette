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
