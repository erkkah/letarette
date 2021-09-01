// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"

	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"

	"github.com/erkkah/letarette/pkg/logger"
)

var cmdline struct {
	Search      bool
	Space       string   `docopt:"<space>"`
	Phrases     []string `docopt:"<phrase>"`
	Limit       int      `docopt:"-l"`
	Offset      int      `docopt:"-p"`
	GroupSize   int32    `docopt:"-g"`
	Interactive bool     `docopt:"-i"`

	Monitor bool

	SQL       bool     `docopt:"sql"`
	Statement string   `docopt:"<sql>"`
	Args      []string `docopt:"<arg>"`

	Index        bool
	Database     string `docopt:"-d"`
	Stats        bool
	Check        bool
	Pgsize       bool
	Compress     bool
	Size         int `docopt:"<size>"`
	Rebuild      bool
	Optimize     bool
	ForceStemmer bool `docopt:"forcestemmer"`

	Load       bool
	JSON       string `docopt:"<json>"`
	AutoAssign bool   `docopt:"-a"`
	LoadLimit  int    `docopt:"-m"`

	Shard string `docopt:"<shard>"`

	Synonyms bool

	Spelling      bool
	Update        bool
	SpellingLimit int `docopt:"<mincount>"`

	ResetMigration bool `docopt:"resetmigration"`
	Version        int  `docopt:"<version>"`

	Env     bool
	Verbose bool `docopt:"-v"`
}

func main() {
	usage := `Letarette

Usage:
    lrcli search [-l <limit>] [-p <page>] [-g <groupsize>] [-i] <space> [<phrase>...]
    lrcli monitor
    lrcli sql [-d <db>] <sql> [<arg>...]
    lrcli index [-d <db>] stats
    lrcli index [-d <db>] check
    lrcli index [-d <db>] pgsize <size>
    lrcli index [-d <db>] compress
    lrcli index [-d <db>] optimize
    lrcli index [-d <db>] rebuild
    lrcli index [-d <db>] forcestemmer
    lrcli load [-d <db>] [-m <max>] [-a] <space> <json>
    lrcli synonyms [-d <db>] [<json>]
    lrcli spelling [-d <db>] update <mincount>
    lrcli resetmigration [-d <db>] <version>
    lrcli env [-v]

Options:
    -l <limit>     Search result page limit [default: 10]
    -p <page>      Search result page [default: 0]
    -d <db>        Override default or environment DB path
    -i             Interactive search
    -a             Auto-assign document ID on load
	-m <max>       Max documents loaded
    -g <groupsize> Force shard group size, do not discover
    -v             Verbose, lists advanced options
`

	args, err := docopt.ParseDoc(usage)
	if err != nil {
		logger.Error.Printf("Failed to parse args: %v", err)
		return
	}

	err = args.Bind(&cmdline)
	if err != nil {
		logger.Error.Printf("Failed to bind args: %v", err)
		return
	}

	cfg, err := letarette.LoadConfig()
	if err != nil {
		logger.Error.Printf("Config load problems: %v", err)
		return
	}
	cfg.DB.ToolConnection = true
	if cmdline.Database != "" {
		cfg.DB.Path = cmdline.Database
	}
	cfg.Index.Spaces = []string{}

	switch {
	case cmdline.Env:
		letarette.Usage(cmdline.Verbose)
	case cmdline.Search:
		doSearch(cfg)
	case cmdline.Index:
		dbSubcommand(cfg)
	case cmdline.Load:
		cfg.Index.Spaces = []string{cmdline.Space}
		logger.Debug.Printf("Loading into space %v", cfg.Index.Spaces)
		dbSubcommand(cfg)
	case cmdline.Synonyms:
		dbSubcommand(cfg)
	case cmdline.Spelling:
		updateSpelling(cfg)
	case cmdline.ResetMigration:
		resetMigration(cfg, cmdline.Version)
	case cmdline.SQL:
		doSQL(cfg)
	case cmdline.Monitor:
		doMonitor(cfg)
	}
}

func dbSubcommand(cfg letarette.Config) {
	db, err := letarette.OpenDatabase(cfg)
	defer func() {
		if db == nil {
			return
		}
		logger.Debug.Printf("Closing db...")
		err := db.Close()
		if err != nil {
			logger.Error.Printf("Failed to close db: %v", err)
		}
	}()

	if err != nil {
		logger.Error.Printf("Failed to open db: %v", err)
		return
	}

	switch {
	case cmdline.Load:
		bulkLoad(db, int(cfg.ShardgroupSize), int(cfg.ShardIndex))
	case cmdline.Synonyms:
		if cmdline.JSON != "" {
			loadSynonyms(db)
		} else {
			dumpSynonyms(db)
		}
	case cmdline.Check:
		err = letarette.CheckStemmerSettings(db, cfg)
		if errors.Is(err, letarette.ErrStemmerSettingsMismatch) {
			logger.Warning.Printf("Index and config stemmer settings mismatch. Re-build index or force changes.")
		}
		checkIndex(db)
	case cmdline.Compress:
		compressIndex(db)
	case cmdline.Pgsize:
		setIndexPageSize(db, cmdline.Size)
	case cmdline.Stats:
		printIndexStats(db)
	case cmdline.Optimize:
		optimizeIndex(db)
	case cmdline.Rebuild:
		rebuildIndex(db)
	case cmdline.ForceStemmer:
		settings := snowball.Settings{
			Stemmers:         cfg.Stemmer.Languages,
			RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
			Separators:       cfg.Stemmer.Separators,
			TokenCharacters:  cfg.Stemmer.TokenCharacters,
		}
		forceIndexStemmerState(settings, db)
	}
}
