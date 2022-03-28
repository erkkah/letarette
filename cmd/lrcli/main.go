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
	"fmt"
	"os"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/pennant"
)

type globalOptions struct {
	Verbose bool `name:"v"`
}

type databaseOptions struct {
	globalOptions
	Database string `name:"d"`
}

func usage() {
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
	fmt.Println(usage)
	os.Exit(1)
}

func main() {

	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	cfg, err := letarette.LoadConfig()
	if err != nil {
		logger.Error.Printf("Config load problems: %v", err)
		return
	}
	cfg.DB.ToolConnection = true
	cfg.Index.Spaces = []string{}

	updateFromFromOptions := func(options *databaseOptions) {
		if options.Database != "" {
			cfg.DB.Path = options.Database
		}
	}

	switch cmd {
	case "search":
		{
			var options searchOptions
			pennant.MustParse(&options, args)
			doSearch(cfg, options)
		}
	case "env":
		{
			var options globalOptions
			pennant.MustParse(&options, args)
			letarette.Usage(options.Verbose)
		}
	case "index":
		{
			var options indexOptions
			pennant.MustParse(&options, args)
			updateFromFromOptions(&options.databaseOptions)
			indexSubcommand(cfg, options)
		}
	case "load":
		{
			var options bulkLoadOptions
			pennant.MustParse(&options, args)
			updateFromFromOptions(&options.databaseOptions)

			if len(options.Space) == 0 || len(options.JSON) == 0 {
				usage()
			}

			cfg.Index.Spaces = []string{options.Space}
			logger.Debug.Printf("Loading into space %v", cfg.Index.Spaces)
			doLoad(cfg, options)
		}
	case "synonyms":
		{
			var options synonymOptions
			pennant.MustParse(&options, args)
			updateFromFromOptions(&options.databaseOptions)
			doSynonyms(cfg, options)
		}
	case "spelling":
		{
			var options spellingOptions
			pennant.MustParse(&options, args)
			updateFromFromOptions(&options.databaseOptions)
			if options.Command != "update" {
				usage()
			}
			if options.MinCount <= 1 {
				usage()
			}
			updateSpelling(cfg, options.MinCount)
		}

	case "resetmigration":
		{
			var options migrationOptions
			pennant.MustParse(&options, args)
			if options.Version < 1 {
				usage()
			}
			updateFromFromOptions(&options.databaseOptions)
			resetMigration(cfg, options.Version)
		}
	case "sql":
		{
			var options sqlOptions
			pennant.MustParse(&options, args)
			updateFromFromOptions(&options.databaseOptions)
			doSQL(cfg, options.Statement, options.Args)
		}
	case "monitor":
		doMonitor(cfg)
	default:
		usage()
	}
}

type indexOptions struct {
	databaseOptions
	Subcommand string `arg:"0"`
	Size       int    `arg:"1"`
}

type scopedDatabase struct {
	db letarette.Database
}

func openDatabase(cfg letarette.Config) (scopedDatabase, error) {
	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		return scopedDatabase{}, err
	}
	return scopedDatabase{db}, nil
}

func (db scopedDatabase) close() {
	if db.db == nil {
		return
	}
	logger.Debug.Printf("Closing db...")
	err := db.db.Close()
	if err != nil {
		logger.Error.Printf("Failed to close db: %v", err)
	}
}

func doLoad(cfg letarette.Config, options bulkLoadOptions) {
	scoped, err := openDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to open db: %v", err)
		return
	}
	defer scoped.close()
	db := scoped.db

	bulkLoad(db, options, int(cfg.ShardgroupSize), int(cfg.ShardIndex))
}

type synonymOptions struct {
	databaseOptions
	File string `arg:"0"`
}

func doSynonyms(cfg letarette.Config, options synonymOptions) {
	scoped, err := openDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to open db: %v", err)
		return
	}
	defer scoped.close()
	db := scoped.db

	if options.File != "" {
		loadSynonyms(db, options.File)
	} else {
		dumpSynonyms(db)
	}
}

func indexSubcommand(cfg letarette.Config, options indexOptions) {
	scoped, err := openDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to open db: %v", err)
		return
	}
	defer scoped.close()
	db := scoped.db

	switch options.Subcommand {

	case "check":
		err = letarette.CheckStemmerSettings(db, cfg)
		if errors.Is(err, letarette.ErrStemmerSettingsMismatch) {
			logger.Warning.Printf("Index and config stemmer settings mismatch. Re-build index or force changes.")
		}
		checkIndex(db)
	case "compress":
		compressIndex(db)
	case "pgsize":
		setIndexPageSize(db, options.Size)
	case "stats":
		printIndexStats(db)
	case "optimize":
		optimizeIndex(db)
	case "rebuild":
		rebuildIndex(db)
	case "forcestemmer":
		settings := snowball.Settings{
			Stemmers:         cfg.Stemmer.Languages,
			RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
			Separators:       cfg.Stemmer.Separators,
			TokenCharacters:  cfg.Stemmer.TokenCharacters,
		}
		forceIndexStemmerState(settings, db)
	default:
		usage()
	}
}
