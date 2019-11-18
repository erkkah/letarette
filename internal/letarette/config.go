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
	"strconv"
	"strings"
	"time"

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
		Spaces                  []string      `required:"true" default:"docs"`
		ChunkSize               uint16        `default:"250"`
		MaxInterestWait         time.Duration `default:"5s"`
		DocumentRefetchInterval time.Duration `default:"1s"`
		MaxDocumentWait         time.Duration `default:"20s"`
		CycleWait               time.Duration `default:"100ms"`
		EmptyCycleWait          time.Duration `default:"4s"`
		MaxOutstanding          uint16        `default:"25"`
		Disable                 bool          `default:"false"`
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
		Strategy       int           `default:"1"`
	}
	Shardgroup      string `default:"1/1"`
	ShardgroupSize  uint16 `ignored:"true"`
	ShardgroupIndex uint16 `ignored:"true"`
	MetricsPort     uint16 `split_words:"true" default:"8000"`
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
	return (cfg.Index.MaxInterestWait > time.Millisecond*20 &&
		cfg.Index.CycleWait > time.Millisecond &&
		cfg.Index.CycleWait < cfg.Index.EmptyCycleWait &&
		cfg.Index.DocumentRefetchInterval > time.Millisecond*20 &&
		cfg.Index.DocumentRefetchInterval < cfg.Index.MaxDocumentWait)
}

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	envconfig.Usage(prefix, &cfg)
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
