// Copyright 2019 Erik Agsjö
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

package letarette

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/erkkah/letarette/pkg/charma"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URL         string `default:"nats://localhost:4222"`
		Topic       string `default:"leta"`
		SearchGroup string `ignored:"true"`
	}
	Db struct {
		Path           string `default:"letarette.db"`
		ToolConnection bool   `ignored:"true"`
	}
	Index struct {
		Spaces         []string `required:"true" default:"docs"`
		ChunkSize      uint16   `default:"250"`
		MaxOutstanding uint16   `split_words:"true" default:"25"`
		Wait           struct {
			Interest        time.Duration `default:"5s"`
			DocumentRefetch time.Duration `default:"1s"`
			Document        time.Duration `default:"20s"`
			Cycle           time.Duration `default:"100ms"`
			EmptyCycle      time.Duration `default:"4s"`
		}
		Disable bool `default:"false"`
	}
	Spelling struct {
		MinFrequency int `split_words:"true" default:"5"`
		MaxLag       int `split_words:"true" default:"100"`
	}
	Stemmer struct {
		Languages        []string `split_words:"true" required:"true" default:"english"`
		RemoveDiacritics bool     `split_words:"true" default:"true"`
		TokenCharacters  string
		Separators       string
	}
	Search struct {
		Timeout        time.Duration `default:"500ms"`
		Cap            int           `default:"25000"`
		CacheTimeout   time.Duration `split_words:"true" default:"1m"`
		CacheMaxsizeMB uint64        `split_words:"true" default:"250"`
		Disable        bool          `default:"false"`
		Strategy       int           `default:"1" desc:"internal"`
	}
	Shardgroup      string `default:"1/1"`
	ShardgroupSize  uint16 `ignored:"true"`
	ShardgroupIndex uint16 `ignored:"true"`
	MetricsPort     uint16 `split_words:"true" default:"8000" desc:"internal"`
}

const prefix = "LETARETTE"

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

	if len(cfg.Index.Spaces) < 1 {
		return Config{}, fmt.Errorf("No spaces defined")
	}

	unique := map[string]string{}
	for _, v := range cfg.Index.Spaces {
		unique[v] = v
	}
	if len(unique) != len(cfg.Index.Spaces) {
		return Config{}, fmt.Errorf("Space names must be unique")
	}

	if !validateIndexDurations(cfg) {
		return Config{}, fmt.Errorf("Invalid index timing settings")
	}

	group, size, err := parseShardGroupString(cfg.Shardgroup)
	if err != nil {
		return
	}
	cfg.ShardgroupIndex = uint16(group - 1)
	cfg.ShardgroupSize = uint16(size)

	cfg.Nats.SearchGroup = fmt.Sprintf("%v", cfg.ShardgroupIndex)
	return
}

func validateIndexDurations(cfg Config) bool {
	return (cfg.Index.Wait.Interest > time.Millisecond*20 &&
		cfg.Index.Wait.Cycle > time.Millisecond &&
		cfg.Index.Wait.Cycle < cfg.Index.Wait.EmptyCycle &&
		cfg.Index.Wait.DocumentRefetch > time.Millisecond*20 &&
		cfg.Index.Wait.DocumentRefetch < cfg.Index.Wait.Document)
}

var usageFormat = "{{$t:=\"\t\"}}" + charma.CircleChars("Letarette") + `
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
	envconfig.Usagef(prefix, &cfg, tabs, usageFormat)
}

func parseShardGroupString(shardGroup string) (group, size int, err error) {
	parts := strings.SplitN(shardGroup, "/", 2)
	parseError := fmt.Errorf("Invalid shard group setting")
	if len(parts) != 2 {
		err = parseError
		return
	}
	group, err = strconv.Atoi(parts[0])
	if err != nil {
		err = parseError
		return
	}
	size, err = strconv.Atoi(parts[1])
	if err != nil {
		err = parseError
		return
	}
	if group > size || group < 1 {
		err = parseError
	}
	return
}
