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

/*
"bygg" is an attempt to replace the roles of "make" and "bash" in building
go projects, making it easier to maintain a portable build environment.
*/
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"
)

var config struct {
	verbose bool
	dryRun  bool
	baseDir string
	byggFil string
}

func main() {
	flag.StringVar(&config.byggFil, "f", "byggfil", "Bygg file")
	flag.BoolVar(&config.dryRun, "n", false, "Performs a dry run")
	flag.BoolVar(&config.verbose, "v", false, "Verbose")
	flag.StringVar(&config.baseDir, "C", ".", "Base dir")
	flag.Parse()

	tgt := "all"

	args := flag.Args()
	if len(args) > 0 {
		tgt = args[0]
	}

	verbose("Building target %q", tgt)

	b, err := newBygg(config.baseDir)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	err = b.buildTarget(tgt)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

type target struct {
	name          string
	buildCommands []string
	dependencies  []string
	resolved      bool
	force         bool
	modifiedAt    time.Time
}

type bygge struct {
	lastError error

	targets map[string]target
	vars    map[string]string
	env     map[string]string
	visited map[string]bool
	tmpl    *template.Template
	dir     string
}

func newBygg(dir string) (*bygge, error) {
	pwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		return nil, err
	}
	defer os.Chdir(pwd)

	result := &bygge{
		targets: map[string]target{},
		vars:    map[string]string{},
		env:     map[string]string{},
		visited: map[string]bool{},
		dir:     dir,
	}

	for _, pair := range os.Environ() {
		parts := strings.Split(pair, "=")
		result.env[parts[0]] = parts[1]
	}

	getFunctions := func(b *bygge) template.FuncMap {
		return template.FuncMap{
			"exec": func(prog string, args ...string) string {
				cmd := exec.Command(prog, args...)
				cmd.Env = b.envList()
				var output []byte
				output, b.lastError = cmd.Output()
				return string(output)
			},
			"ok": func() bool {
				return b.lastError == nil
			},
			"date": func(layout string) string {
				return time.Now().Format(layout)
			},
			"split": func(unsplit string) []string {
				return strings.Split(unsplit, " ")
			},
		}
	}

	result.tmpl = template.New(config.byggFil)
	result.tmpl.Funcs(getFunctions(result))

	verbose("Parsing template")
	if !exists(config.byggFil) {
		return nil, fmt.Errorf("bygg file %q not found", config.byggFil)
	}
	var err error

	if result.tmpl, err = result.tmpl.ParseFiles(config.byggFil); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return result, nil
}

func (b *bygge) buildTarget(tgt string) error {
	pwd, _ := os.Getwd()
	if err := os.Chdir(b.dir); err != nil {
		return err
	}
	defer os.Chdir(pwd)

	goVersion := runtime.Version()

	data := map[string]interface{}{
		"GOVERSION": goVersion,
		"env":       b.env,
	}

	verbose("Executing template")
	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, data); err != nil {
		return err
	}

	verbose("Loading build script")
	if err := b.loadBuildScript(&buf); err != nil {
		return err
	}

	if tgt, ok := b.targets[tgt]; ok {
		if err := b.resolve(tgt); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("no such target %q", tgt)
}

func (b *bygge) loadBuildScript(scriptSource io.Reader) error {
	scanner := bufio.NewScanner(scriptSource)

	// Handle dependencies, build commands and assignments, with
	// or without spaces around the operators.
	//
	// Examples:
	// all: foo splat
	// all <- gcc -o all all.c
	// bar=baz
	// bar += yes
	commandExp := regexp.MustCompile(`([\w._\-/${}]+)\s*([:=]|\+=|<-|<<)\s*(.*)`)

	for scanner.Scan() {
		line := scanner.Text()
		// Skip initial whitespace
		line = strings.TrimLeft(line, " \t")
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Skip empty lines
		if line == "" {
			continue
		}
		// Handle message lines
		if strings.HasPrefix(line, "<<") {
			fmt.Println(b.expand(strings.Trim(line[2:], " \t")))
			continue
		}

		matches := commandExp.FindStringSubmatch(line)
		if matches == nil {
			return fmt.Errorf("parse error: %q", line)
		}

		lvalue := matches[1]
		operator := matches[2]
		rvalue := matches[3]

		lvalue = b.expand(lvalue)
		rvalue = b.expand(rvalue)

		var err error
		switch operator {
		case ":":
			err = b.handleDependencies(lvalue, rvalue)
		case "=":
			err = b.handleAssignment(lvalue, rvalue, false)
		case "+=":
			err = b.handleAssignment(lvalue, rvalue, true)
		case "<<":
			rvalue = operator + " " + rvalue
			fallthrough
		case "<-":
			b.handleBuildCommand(lvalue, rvalue)
		default:
			return fmt.Errorf("unexpected operator %q", operator)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *bygge) handleDependencies(lvalue, rvalue string) error {
	t := b.targets[lvalue]
	t.name = lvalue
	rvalue = strings.TrimLeft(rvalue, " \t")
	if strings.HasPrefix(rvalue, "!") {
		t.force = true
		rvalue = strings.TrimLeft(rvalue, "!")
	}
	dependencies, err := splitQuoted(rvalue)
	if err != nil {
		return err
	}
	t.dependencies = append(t.dependencies, dependencies...)
	b.targets[lvalue] = t

	return nil
}

func (b *bygge) handleAssignment(lvalue, rvalue string, add bool) error {
	if strings.Contains(lvalue, ".") {
		parts := strings.SplitN(lvalue, ".", 2)
		context := parts[0]
		name := parts[1]
		if context == "env" {
			if oldValue, isSet := b.env[name]; isSet && add {
				rvalue = oldValue + " " + rvalue
			}
			b.env[name] = rvalue
		} else {
			return fmt.Errorf("unknown variable context %q", context)
		}
	} else {
		if add {
			rvalue = b.vars[lvalue] + " " + rvalue
		}
		b.vars[lvalue] = rvalue
	}

	return nil
}

func (b *bygge) handleBuildCommand(lvalue, rvalue string) {
	t := b.targets[lvalue]
	t.name = lvalue
	t.buildCommands = append(t.buildCommands, rvalue)
	b.targets[lvalue] = t
}

// Permissive variable expansion
func (b *bygge) expand(expr string) string {
	return os.Expand(expr, func(varExpr string) string {
		varExpr = strings.Trim(varExpr, " \t")
		if strings.Contains(varExpr, ".") {
			parts := strings.SplitN(varExpr, ".", 2)
			context := parts[0]
			name := parts[1]

			if context == "env" {
				if local, ok := b.env[name]; ok {
					return local
				}
				return os.Getenv(name)

			}
			return ""
		}
		return b.vars[varExpr]
	})
}

func (b *bygge) resolve(t target) error {
	if t.resolved {
		return nil
	}

	verbose("Resolving target %q", t.name)
	if b.visited[t.name] {
		return fmt.Errorf("cyclic dependency resolving %q", t.name)
	}
	b.visited[t.name] = true
	defer func() {
		b.visited[t.name] = false
	}()

	dependencies := t.dependencies

	var mostRecentUpdate time.Time

	for _, depName := range dependencies {
		dep, ok := b.targets[depName]
		if !ok {
			if exists(depName) {
				dep = target{
					name: depName,
				}
			} else {
				return fmt.Errorf("target %q has unknown dependency %q", t.name, depName)
			}
		}
		if err := b.resolve(dep); err != nil {
			return err
		}
		dep = b.targets[depName]
		if dep.modifiedAt.After(mostRecentUpdate) {
			mostRecentUpdate = dep.modifiedAt
		}
	}

	if t.force || !exists(t.name) || getFileDate(t.name).Before(mostRecentUpdate) {
		for _, cmd := range t.buildCommands {
			if err := b.build(t.name, cmd); err != nil {
				return err
			}
		}
	}

	t.resolved = true

	if exists(t.name) {
		t.modifiedAt = getFileDate(t.name)
	} else {
		t.modifiedAt = time.Now()
	}

	b.targets[t.name] = t

	return nil
}

func (b *bygge) build(tgt, command string) error {
	if config.dryRun {
		fmt.Printf("Not running command %q\n", command)
		return nil
	}
	parts, err := splitQuoted(command)
	if err != nil {
		return err
	}
	prog := parts[0]
	args := parts[1:]
	verbose("Running command %q with args %v", prog, args)
	if prog == "<<" {
		fmt.Println(strings.Join(args, " "))
		return nil
	}
	if prog == "bygg" {
		byggDir := "."
		byggTarget := "all"
		if len(args) > 0 {
			if args[0] == "-C" {
				if len(args) < 2 {
					return fmt.Errorf("invalid bygg arguments")
				}
				byggDir = args[1]
				args = args[2:]
			}
			if len(args) == 2 {
				byggTarget = args[1]
			}
		}
		bb, err := newBygg(byggDir)
		if err != nil {
			return err
		}
		return bb.buildTarget(byggTarget)
	}
	if strings.HasPrefix(prog, "http") {
		return handleDownload(tgt, prog, args...)
	}
	cmd := exec.Command(prog, args...)
	cmd.Env = b.envList()
	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	return err
}

func (b *bygge) envList() []string {
	env := []string{}
	for k, v := range b.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

func verbose(pattern string, args ...interface{}) {
	if config.verbose {
		fmt.Printf("bygg: "+pattern+"\n", args...)
	}
}

func splitQuoted(quoted string) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(quoted))
	scanner.Split(bufio.ScanRunes)

	parts := []string{}

	escapeNext := false
	inString := false

	var builder strings.Builder

	for scanner.Scan() {
		char := scanner.Text()
		switch char {
		case `\`:
			if inString {
				escapeNext = true
			} else {
				builder.WriteString(char)
			}
		case `"`:
			if escapeNext {
				builder.WriteString(char)
			} else {
				inString = !inString
			}
			escapeNext = false
		case ` `:
			if inString {
				builder.WriteString(char)
			} else if builder.Len() != 0 {
				parts = append(parts, builder.String())
				builder.Reset()
			}
			escapeNext = false
		default:
			builder.WriteString(char)
			escapeNext = false
		}
	}
	if inString {
		return parts, fmt.Errorf("unterminated string")
	}
	if builder.Len() != 0 {
		parts = append(parts, builder.String())
	}
	return parts, nil
}

func exists(target string) bool {
	stat, err := os.Stat(target)
	return err == nil && stat != nil
}

func getFileDate(target string) time.Time {
	fileInfo, _ := os.Stat(target)
	if fileInfo == nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}

func handleDownload(target string, url string, checksum ...string) error {
	verbose("Downloading %s", url)
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}

	targetDate := getFileDate(target).In(time.FixedZone("GMT", 0))
	if !targetDate.IsZero() {
		req.Header.Set("If-Modified-Since", targetDate.Format(time.RFC1123))
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		verbose("%s unmodified, skipping download", url)
		return nil
	}

	modified := response.Header.Get("Last-Modified")
	var modificationDate time.Time
	if modified != "" {
		modificationDate, err = time.Parse(time.RFC1123, modified)
		if err != nil {
			modificationDate = time.Time{}
		}
	}

	if err = os.MkdirAll(target, 0770); err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), target)
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err = io.Copy(tmpFile, response.Body); err != nil {
		return err
	}
	_, _ = tmpFile.Seek(0, 0)

	if len(checksum) > 0 && strings.HasPrefix(checksum[0], "md5:") {
		hash := md5.New()
		if _, err = io.Copy(hash, tmpFile); err != nil {
			return err
		}
		sum := fmt.Sprintf("md5:%x", hash.Sum(nil))
		if sum != checksum[0] {
			return fmt.Errorf("checksum verification failed for %q", url)
		}
		_, _ = tmpFile.Seek(0, 0)
	}

	var reader io.Reader = tmpFile
	if strings.HasSuffix(url, "gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return err
		}
	}

	if strings.HasSuffix(url, ".tar") || strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, "tgz") {
		tarReader := tar.NewReader(reader)

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			finfo := hdr.FileInfo()

			switch {
			case finfo.IsDir():
				dir := path.Join(target, hdr.Name)
				if err = os.MkdirAll(dir, finfo.Mode()); err != nil {
					return err
				}
			case finfo.Mode().IsRegular():
				dest, err := os.Create(path.Join(target, hdr.Name))
				if err != nil {
					return err
				}
				if _, err = io.Copy(dest, tarReader); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported file type: %v", finfo.Mode().String())
			}
		}
	} else {
		return fmt.Errorf("unsupported file: %v", url)
	}

	if !modificationDate.IsZero() {
		_ = os.Chtimes(target, modificationDate, modificationDate)
	}

	return nil
}
