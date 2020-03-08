/*
"bygg" is an attempt to replace the roles of "make" and "bash" in building
letarette, making it easier to keep a portable build environment working.

It uses only go builtins and is small enough to be run using "go run".
*/
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
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
}

func main() {
	flag.BoolVar(&config.dryRun, "n", false, "Performs a dry run")
	flag.BoolVar(&config.verbose, "v", false, "Verbose")
	flag.Parse()

	mainTarget := "all"

	args := flag.Args()
	if len(args) > 1 {
		mainTarget = args[1]
	}

	script := args[0]

	verbose("Building target %q from file %q", mainTarget, script)

	tmpl := template.New(path.Base(script))
	tmpl.Funcs(functions)

	verbose("Parsing template")
	tmpl, err := tmpl.ParseFiles(script)

	if err != nil {
		fmt.Printf("Failed to parse templates: %v\n", err)
		os.Exit(1)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		fmt.Printf("Failed to get user cache dir: %v\n", err)
		os.Exit(1)
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
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	verbose("Loading build script")
	err = loadBuildScript(&buf)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	if config.verbose {
		fmt.Println("bygg: Vars:")
		for k, v := range vars {
			fmt.Printf("\t%s=%s\n", k, v)
		}
		fmt.Println("bygg: Targets:")
		for k, v := range targets {
			fmt.Printf("\t%s=%v\n", k, v.dependencies)
		}
	}

	if tgt, ok := targets[mainTarget]; ok {
		err = resolve(tgt)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	} else {
		fmt.Printf("No such target %q", mainTarget)
		os.Exit(1)
	}
}

var lastError error

var functions template.FuncMap = template.FuncMap{
	"exec": func(prog string, args ...string) string {
		cmd := exec.Command(prog, args...)
		cmd.Env = combinedEnv()
		var output []byte
		output, lastError = cmd.Output()
		return string(output)
	},
	"ok": func() bool {
		return lastError == nil
	},
	"date": func(layout string) string {
		return time.Now().Format(layout)
	},
}

func loadBuildScript(scriptSource io.Reader) error {
	scanner := bufio.NewScanner(scriptSource)

	// Handle dependencies, build commands and assignments, with
	// or without spaces around the operators.
	//
	// Examples:
	// all: foo splat
	// all <- gcc -o all all.c
	// bar=baz
	// bar += yes
	commandExp := regexp.MustCompile(`([A-Za-z._\-/${}]+)\s*([:=]|\+=|<-)\s*(.*)`)

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

		lvalue = expand(lvalue)
		rvalue = expand(rvalue)

		var err error
		switch operator {
		case ":":
			err = handleDependencies(lvalue, rvalue)
		case "=":
			err = handleAssignment(lvalue, rvalue, false)
		case "+=":
			err = handleAssignment(lvalue, rvalue, true)
		case "<-":
			err = handleBuildCommand(lvalue, rvalue)
		default:
			return fmt.Errorf("Unexpected operator %q", operator)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type target struct {
	name          string
	buildCommands []string
	dependencies  []string
	resolved      bool
	modifiedAt    time.Time
}

var targets = map[string]target{}
var vars = map[string]string{}
var env = map[string]string{}

func handleDependencies(lvalue, rvalue string) error {
	t := targets[lvalue]
	t.name = lvalue
	dependencies, err := splitQuoted(rvalue)
	if err != nil {
		return err
	}
	t.dependencies = append(t.dependencies, dependencies...)
	targets[lvalue] = t

	return nil
}

func handleAssignment(lvalue, rvalue string, add bool) error {
	if strings.Contains(lvalue, ".") {
		parts := strings.SplitN(lvalue, ".", 2)
		context := parts[0]
		name := parts[1]
		if context == "env" {
			if add {
				rvalue = env[name] + " " + rvalue
			}
			env[name] = rvalue
		} else {
			return fmt.Errorf("Unknown variable context %q", context)
		}
	} else {
		if add {
			rvalue = vars[lvalue] + " " + rvalue
		}
		vars[lvalue] = rvalue
	}

	return nil
}

func handleBuildCommand(lvalue, rvalue string) error {
	t := targets[lvalue]
	t.name = lvalue
	t.buildCommands = append(t.buildCommands, rvalue)
	targets[lvalue] = t

	return nil
}

// Permissive variable expansion
func expand(expr string) string {
	return os.Expand(expr, func(varExpr string) string {
		varExpr = strings.Trim(varExpr, " \t")
		if strings.Contains(varExpr, ".") {
			parts := strings.SplitN(varExpr, ".", 2)
			context := parts[0]
			name := parts[1]

			switch context {
			case "env":
				if local, ok := env[name]; ok {
					return local
				}
				return os.Getenv(name)
			default:
				return ""
			}
		} else {
			return vars[varExpr]
		}
	})
}

var visited = map[string]bool{}

func resolve(t target) error {
	if t.resolved {
		return nil
	}

	verbose("Resolving target %q", t.name)
	if visited[t.name] {
		return fmt.Errorf("Cyclic dependency resolving %q", t.name)
	}
	visited[t.name] = true
	defer func() {
		visited[t.name] = false
	}()

	dependencies := t.dependencies

	var mostRecentUpdate time.Time

	for _, depName := range dependencies {
		dep, ok := targets[depName]
		if !ok {
			if targetExists(depName) {
				dep = target{
					name: depName,
				}
			} else {
				return fmt.Errorf("Target %q has unknown dependency %q", t.name, depName)
			}
		}
		err := resolve(dep)
		if err != nil {
			return err
		}
		if dep.modifiedAt.After(mostRecentUpdate) {
			mostRecentUpdate = dep.modifiedAt
		}
	}

	if !targetExists(t.name) || mostRecentUpdate.IsZero() || getTargetDate(t.name).Before(mostRecentUpdate) {
		for _, cmd := range t.buildCommands {
			err := build(cmd)
			if err != nil {
				return err
			}
		}
	}

	t.resolved = true

	if targetExists(t.name) {
		t.modifiedAt = getTargetDate(t.name)
	} else {
		t.modifiedAt = time.Now()
	}
	return nil
}

func build(command string) error {
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
	cmd := exec.Command(prog, args...)
	cmd.Env = combinedEnv()
	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	return err
}

func combinedEnv() []string {
	localEnv := []string{}
	for k, v := range env {
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
				parts = append(parts, builder.String())
				builder.Reset()
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

func targetExists(target string) bool {
	_, err := os.Stat(target)
	return !os.IsNotExist(err)
}

func getTargetDate(target string) time.Time {
	fileInfo, _ := os.Stat(target)
	if fileInfo == nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}
