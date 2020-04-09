package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/erkkah/letarette"
	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URLS     []string `default:"nats://localhost:4222"`
		SeedFile string
		RootCAs  []string
		Topic    string `default:"leta"`
	}
	Host string `default:""`
	Port uint16 `default:"8080"`
}

const prefix = "LRMON"

// LoadConfig loads configuration variables from the environment
// and returns a fully populated Config instance.
func LoadConfig() (cfg Config, err error) {
	err = envconfig.CheckDisallowed(prefix, &cfg)
	if err != nil {
		return
	}

	err = envconfig.Process(prefix, &cfg)
	if err != nil {
		return
	}

	return
}

var usageFormat = fmt.Sprintf("{{$t:=\"\t\"}}Letarette monitor\n%s (%s)\n", letarette.Tag, letarette.Revision) + `
Configuration environment variables:

VARIABLE{{$t}}TYPE{{$t}}DEFAULT
========{{$t}}===={{$t}}=======
LOG_LEVEL{{$t}}String{{$t}}INFO
{{range .}}{{if usage_description . | eq "internal" | not}}{{usage_key .}}{{$t}}{{usage_type .}}{{$t}}{{usage_default .}}
{{end}}{{end}}
`

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	_ = envconfig.Usagef(prefix, &cfg, tabs, usageFormat)
}
