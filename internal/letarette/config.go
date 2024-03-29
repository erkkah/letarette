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
	"text/tabwriter"
	"time"

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
	DB struct {
		Path           string `default:"letarette.db"`
		CacheSizeMB    uint32 `default:"1024" desc:"advanced"` // default 1G DB cache
		MMapSizeMB     uint32 `default:"0" desc:"internal"`    // no DB mmap by default
		ToolConnection bool   `ignored:"true"`
	}
	Index struct {
		Spaces         []string `required:"true" default:"docs"`
		ListSize       uint16   `default:"500" desc:"advanced"`
		ReqSize        uint16   `default:"50" desc:"advanced"`
		MaxOutstanding uint16   `split_words:"true" default:"4" desc:"advanced"`
		Wait           struct {
			Cycle      time.Duration `default:"100ms" desc:"advanced"`
			EmptyCycle time.Duration `default:"5s" desc:"advanced"`
			Interest   time.Duration `default:"5s" desc:"advanced"`
			Document   time.Duration `default:"30s" desc:"advanced"`
			Refetch    time.Duration `default:"3s" desc:"advanced"`
		}
		Disable  bool `default:"false" desc:"advanced"`
		Compress bool `default:"false"`
	}
	Spelling struct {
		MinFrequency int `split_words:"true" default:"5" desc:"advanced"`
		MaxLag       int `split_words:"true" default:"100" desc:"advanced"`
	}
	Stemmer struct {
		Languages        []string `split_words:"true" required:"true" default:"english"`
		RemoveDiacritics bool     `split_words:"true" default:"true" desc:"advanced"`
		TokenCharacters  string   `desc:"advanced"`
		Separators       string   `desc:"advanced"`
		StopwordCutoff   float32  `split_words:"true" default:"1" desc:"advanced"`
	}
	Search struct {
		Timeout        time.Duration `default:"4s"`
		Cap            int           `default:"10000"`
		CacheTimeout   time.Duration `split_words:"true" default:"10m"`
		CacheMaxsizeMB uint64        `split_words:"true" default:"250"`
		Disable        bool          `default:"false" desc:"advanced"`
		Strategy       int           `default:"1" desc:"internal"`
	}
	Shard          string `default:"1/1"`
	ShardgroupSize uint16 `ignored:"true"`
	ShardIndex     uint16 `ignored:"true"`
	CloningPort    uint16 `default:"8192"`
	CloningHost    string
	Profile        struct {
		HTTP  int    `desc:"internal"`
		CPU   string `desc:"internal"`
		Mem   string `desc:"internal"`
		Block string `desc:"internal"`
		Mutex string `desc:"internal"`
	}
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
		return Config{}, fmt.Errorf("no spaces defined")
	}

	unique := map[string]string{}
	for _, v := range cfg.Index.Spaces {
		unique[v] = v
	}
	if len(unique) != len(cfg.Index.Spaces) {
		return Config{}, fmt.Errorf("space names must be unique")
	}

	if !validateIndexDurations(cfg) {
		return Config{}, fmt.Errorf("invalid index timing settings")
	}

	group, size, err := parseShardString(cfg.Shard)
	if err != nil {
		return
	}
	cfg.ShardIndex = uint16(group - 1)
	cfg.ShardgroupSize = uint16(size)

	return
}

func validateIndexDurations(cfg Config) bool {
	return (cfg.Index.Wait.Interest > time.Millisecond*20 &&
		cfg.Index.Wait.Cycle < cfg.Index.Wait.EmptyCycle &&
		cfg.Index.Wait.Refetch > time.Millisecond*20 &&
		cfg.Index.Wait.Refetch < cfg.Index.Wait.Document)
}

var usageFormat = fmt.Sprintf(
	"{{$t:=\"\t\"}}Letarette\n%s\n",
	letarette.Version(),
) + `
Configuration environment variables:

VARIABLE{{$t}}TYPE{{$t}}DEFAULT
========{{$t}}===={{$t}}=======
LOG_LEVEL{{$t}}String{{$t}}INFO
{{range .}}{{if usage_description . | eq ""}}{{usage_key .}}{{$t}}{{usage_type .}}{{$t}}{{usage_default .}}
{{end}}{{end}}
`

var advancedFormat = `
Advanced configuration:

VARIABLE{{$t}}TYPE{{$t}}DEFAULT
========{{$t}}===={{$t}}=======
{{range .}}{{if usage_description . | eq "advanced"}}{{usage_key .}}{{$t}}{{usage_type .}}{{$t}}{{usage_default .}}
{{end}}{{end}}
`

// Usage prints usage help to stdout
func Usage(advanced bool) {
	var cfg Config
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	format := usageFormat
	if advanced {
		format += advancedFormat
	}
	_ = envconfig.Usagef(prefix, &cfg, tabs, format)
}
