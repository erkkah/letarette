/*
"bygg" is an attempt to replace the roles of "make" and "bash" in building
letarette, making it easier to keep a portable build environment working.

It uses only go builtins and is small enough to be run using "go run".
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
	"path/filepath"
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
}

func main() {
	flag.BoolVar(&config.dryRun, "n", false, "Performs a dry run")
	flag.BoolVar(&config.verbose, "v", false, "Verbose")
	flag.StringVar(&config.baseDir, "C", ".", "Base dir")
	flag.Parse()

	tgt := "all"

	args := flag.Args()
	if len(args) > 1 {
		tgt = args[1]
	}

	script := args[0]

	verbose("Building target %q from file %q", tgt, script)

	b, err := newBygg(config.baseDir, script)
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

func newBygg(dir, script string) (*bygge, error) {
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

	getFunctions := func(b *bygge) template.FuncMap {
		return template.FuncMap{
			"exec": func(prog string, args ...string) string {
				cmd := exec.Command(prog, args...)
				cmd.Env = b.combinedEnv()
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

	result.tmpl = template.New(path.Base(script))
	result.tmpl.Funcs(getFunctions(result))

	verbose("Parsing template %q", script)
	if !exists(script) {
		return nil, fmt.Errorf("Bygg file %q not found", script)
	}
	var err error
	result.tmpl, err = result.tmpl.ParseFiles(script)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse templates: %w", err)
	}
	return result, nil
}

func (b *bygge) buildTarget(tgt string) error {
	pwd, _ := os.Getwd()
	if err := os.Chdir(b.dir); err != nil {
		return err
	}
	defer os.Chdir(pwd)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("Failed to get user cache dir: %v", err)
	}
	goCache := filepath.Join(cacheDir, "go-build")
	goVersion := runtime.Version()

	env := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.Split(pair, "=")
		env[parts[0]] = parts[1]
	}

	data := map[string]interface{}{
		"GO_CACHE":   goCache,
		"GO_VERSION": goVersion,
		"env":        env,
	}

	verbose("Executing template")
	var buf bytes.Buffer
	err = b.tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	verbose("Loading build script")
	err = b.loadBuildScript(&buf)
	if err != nil {
		return err
	}

	if config.verbose {
		fmt.Println("bygg: Vars:")
		for k, v := range b.vars {
			fmt.Printf("\t%s=%s\n", k, v)
		}
		fmt.Println("bygg: Targets:")
		for k, v := range b.targets {
			fmt.Printf("\t%s=%v\n", k, v.dependencies)
		}
	}

	if tgt, ok := b.targets[tgt]; ok {
		err = b.resolve(tgt)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("No such target %q", tgt)
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
	commandExp := regexp.MustCompile(`([\w._\-/${}]+)\s*([:=]|\+=|<-)\s*(.*)`)

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
			fmt.Println(strings.Trim(line[2:], " \t"))
			continue
		}

		matches := commandExp.FindStringSubmatch(line)
		if matches == nil {
			return fmt.Errorf("Parse error: %q", line)
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
		case "<-":
			err = b.handleBuildCommand(lvalue, rvalue)
		default:
			return fmt.Errorf("Unexpected operator %q", operator)
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
			if add {
				rvalue = b.env[name] + " " + rvalue
			}
			b.env[name] = rvalue
		} else {
			return fmt.Errorf("Unknown variable context %q", context)
		}
	} else {
		if add {
			rvalue = b.vars[lvalue] + " " + rvalue
		}
		b.vars[lvalue] = rvalue
	}

	return nil
}

func (b *bygge) handleBuildCommand(lvalue, rvalue string) error {
	t := b.targets[lvalue]
	t.name = lvalue
	t.buildCommands = append(t.buildCommands, rvalue)
	b.targets[lvalue] = t

	return nil
}

// Permissive variable expansion
func (b *bygge) expand(expr string) string {
	return os.Expand(expr, func(varExpr string) string {
		varExpr = strings.Trim(varExpr, " \t")
		if strings.Contains(varExpr, ".") {
			parts := strings.SplitN(varExpr, ".", 2)
			context := parts[0]
			name := parts[1]

			switch context {
			case "env":
				if local, ok := b.env[name]; ok {
					return local
				}
				return os.Getenv(name)
			default:
				return ""
			}
		} else {
			return b.vars[varExpr]
		}
	})
}

func (b *bygge) resolve(t target) error {
	if t.resolved {
		return nil
	}

	verbose("Resolving target %q", t.name)
	if b.visited[t.name] {
		return fmt.Errorf("Cyclic dependency resolving %q", t.name)
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
				return fmt.Errorf("Target %q has unknown dependency %q", t.name, depName)
			}
		}
		err := b.resolve(dep)
		if err != nil {
			return err
		}
		dep = b.targets[depName]
		if dep.modifiedAt.After(mostRecentUpdate) {
			mostRecentUpdate = dep.modifiedAt
		}
	}

	if !exists(t.name) || mostRecentUpdate.IsZero() || getFileDate(t.name).Before(mostRecentUpdate) {
		for _, cmd := range t.buildCommands {
			err := b.build(t.name, cmd)
			if err != nil {
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
	if prog == "bygg" {
		byggDir := "."
		byggTarget := "all"
		if len(args) == 0 {
			return fmt.Errorf("Missing buildfile")
		}
		if args[0] == "-C" {
			if len(args) < 3 {
				return fmt.Errorf("Invalid bygg arguments")
			}
			byggDir = args[1]
			args = args[2:]
		}
		if len(args) == 2 {
			byggTarget = args[1]
		}
		bb, err := newBygg(byggDir, args[0])
		if err != nil {
			return err
		}
		return bb.buildTarget(byggTarget)
	}
	if strings.HasPrefix(prog, "http") {
		return handleDownload(tgt, prog, args...)
	}
	cmd := exec.Command(prog, args...)
	cmd.Env = b.combinedEnv()
	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	return err
}

func (b *bygge) combinedEnv() []string {
	localEnv := []string{}
	for k, v := range b.env {
		localEnv = append(localEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return append(os.Environ(), localEnv...)
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
			escapeNext = true
		case `"`:
			if escapeNext {
				builder.WriteString(char)
				escapeNext = false
			} else {
				inString = !inString
			}
		case ` `:
			if inString {
				builder.WriteString(char)
			} else {
				if builder.Len() != 0 {
					parts = append(parts, builder.String())
					builder.Reset()
				}
			}
		default:
			builder.WriteString(char)
		}
	}
	if inString {
		return parts, fmt.Errorf("Unterminated string")
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
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	err = os.MkdirAll(target, 0770)
	if err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), target)
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	_, err = io.Copy(tmpFile, response.Body)
	if err != nil {
		return err
	}
	tmpFile.Seek(0, 0)

	if len(checksum) > 0 && strings.HasPrefix(checksum[0], "md5:") {
		hash := md5.New()
		_, err = io.Copy(hash, tmpFile)
		if err != nil {
			return err
		}
		sum := fmt.Sprintf("md5:%x", hash.Sum(nil))
		if sum != checksum[0] {
			return fmt.Errorf("Checksum verification failed for %q", url)
		}
		tmpFile.Seek(0, 0)
	}

	var reader io.Reader = tmpFile
	if strings.HasSuffix(url, "gz") {
		reader, err = gzip.NewReader(reader)
		if err != nil {
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

			if finfo.IsDir() {
				dir := path.Join(target, hdr.Name)
				err = os.MkdirAll(dir, finfo.Mode())
				if err != nil {
					return err
				}
			} else if finfo.Mode().IsRegular() {
				dest, err := os.Create(path.Join(target, hdr.Name))
				if err != nil {
					return err
				}
				_, err = io.Copy(dest, tarReader)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Unsupported file type: %v", finfo.Mode().String())
			}
		}
	} else {
		return fmt.Errorf("Unsupported file: %v", url)
	}

	return nil
}
