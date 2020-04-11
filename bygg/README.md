# bygg!

This is an attempt to get a portable way of building [Letarette](https://letarette.io), but it should work well for other `go` projects with similar needs.

Letarette is a `go` project, but since it relies heavily on `sqlite3`, a C compiler is required. Also, the stemming is done by the `Snowball` C library which comes with a Makefile - based build.

I started out using `make` and `bash`, which worked fine for a while.
I went through a couple of iterations before the Linux and Mac builds worked the same, but when I got to Windows, I hit a wall.

So, I started thinking - could I script the build process in `go` ?

As usual, I let it grow a little bit too far. But - it's still just about 600 lines, including comments, and has no external dependencies.

## The tool

The `bygg` tool uses concepts similar to `make`, where the build process is described in a `byggfil` by listing dependencies and build steps. The `byggfil` is preprocessed as a `go` template, with a couple of help functions.

This is obviously not `make`, and admittedly `go` templates are a bit weird, but it works really well for my needs and I've been able to simplify my build process, so I'm happy.

## Running a build

A build is started by running the tool in the directory containing the `byggfil`.
Since it is a single-file no-dependencies tool, running using `go run` is fast enough:

```
$ go run ./bygg
```

In Letarette, there is an `autoexec.go` file at the root of the project with a `go:generate` line that starts the build:

```go
// bygg! entry point, just run "go generate" to build Letarette.

//go:generate go run ./bygg

package letarette
```

The `bygg` tool accepts these arguments:

```
Usage:
  bygg <options> <target>

Options:
  -C string
    	Base dir (default ".")
  -f string
    	Bygg file (default "byggfil")
  -n	Performs a dry run
  -v	Verbose
```

The default target is "all".

## `byggfil` syntax

### Dependencies

*Dependencies* are specified using comma - statements:

```
target: dependency1 dependency2
```

Existing targets will only be rebuilt if any of the depencies are newer, as usual.
For targets that should always be built, or when the dependency analysis is done by the build tool, building can be forced by prefixing the (possibly empty) dependency list with an exclamation mark:

```
target: !
```

### Build commands

*Build commands* for a target are specified using arrow statements:

```
target <- go build .
```

Multiple build commands for a single target are run in the order they appear.
Except for the special cases described below, build commands are external binaries that are run in the currently set environment.

#### Child builds

A child `bygg` build can be run as expected:

```
target <- bygg -C ./path/to/submodule
```

There is nothing stopping you from running endless build loops using child builds.

#### Downloads

If a build command starts with a URL to a `tar`, `tar.gz` or `tgz` file, that file will be downloaded and unpacked into a directory with the name of the target. The download can optionally be verified by an `md5` checksum:

```
lib <- https://where.files.live/mylittle.lib.tgz md5:f8288a861db7c97dc4750020c7c7aa6f
```

> NOTE: Downloads are considered to be up to date if the target directory is not older than the "Last-Modified" header.

#### Logging

The special build operator `<<` prints the rest of the line to stdout:

```
parser-gen << Building the parser generator now!
```

The `<<` operator can also be used by itself, outside of build commands. In this case the output will be printed while interpreting, before targets are resolved.

```
<< Starting build {{date "2006-01-02 15:04:05"}}
```

### Variables

Variables are set and added to like this:

```
CFLAGS = -O2
CFLAGS += -Wall
```

Using `+=` to add a value to a variable will add a space between the old value and the addition.
This can be avoided using variable interpolation.

Environment variables live in the `env` namespace:

```
env.CC = gcc
```

Variable interpolation looks familiar, and works in both lvalues and rvalues:

```
${MY_TARGET}: dependency $OBJFILES
```

### Template execution

Before the build script is interpreted, it is run through the `go` [text template engine.](https://golang.org/pkg/text/template/)
This makes it possible to to do things like:

```
REV = dev

{{if (exec "git" "status" "--porcelain") | eq "" }}
    repoState = Clean repo, tagging with git rev
    REV = {{slice (exec "git" "rev-parse" "HEAD") 0 7 }}
{{else}}
    repoState = Dirty repo, tagging as dev build
{{end}}

<< $repoState
```

The rendering data object is a map containing the environment and current `go` version:

```
<< Running in go version {{.GOVERSION}}
<< PATH is set to {{.env.PATH}}
```

In addition to the [standard functions](https://golang.org/pkg/text/template/#hdr-Functions), `bygg` adds the following: 

#### exec
Returns the output of running the command specified by the first argument, with the rest of the arguments as command line arguments.

#### ok
Returns boolean true if the last `exec` was successful.

#### date
Returns the current date and time, formatted according to its argument.
The format is passed directly to the `go` date formatter:

```
{{date "2006-01-02"}}
```

#### split
Returns a slice of strings by splitting its argument by spaces.

## Syntax highlighting

To make it more fun to edit `bygg` files in VS Code, I put together a basic syntax highlighting package. Search for "bygg syntax highlighting" in the marketplace.

You're welcome!
