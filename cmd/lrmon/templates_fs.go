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

//go:build !prod
// +build !prod

package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
)

func errorTemplate(templatePath string, err error) *template.Template {
	result := template.New(templatePath)
	_, _ = result.Parse(fmt.Sprintf("Template error: %v", err))
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

func serveRaw(rawPath string, writer io.Writer) error {
	file, err := os.Open(path.Join("static", rawPath))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}
