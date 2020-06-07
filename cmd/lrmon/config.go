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
