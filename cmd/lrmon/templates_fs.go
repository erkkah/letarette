// +build !prod

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"path"
)

func errorTemplate(templatePath string, err error) *template.Template {
	result := template.New(templatePath)
	result.Parse(fmt.Sprintf("Template error: %v", err))
	return result
}

func lookupTemplate(templatePath string) *template.Template {

	data, err := ioutil.ReadFile(path.Join("static", templatePath))
	if err != nil {
		return errorTemplate(templatePath, err)
	}
	result := template.New(templatePath).Funcs(templateFunctions)
	_, err = result.Parse(string(data))
	if err != nil {
		return errorTemplate(templatePath, err)
	}
	return result
}
