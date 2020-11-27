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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/erkkah/letarette/pkg/spinner"
)

func checkIndex(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Checking index ")

	err := letarette.CheckIndex(db)
	if err != nil {
		s.Stop(fmt.Sprintf("Index check failed: %v\n", err))
		return
	}

	s.Stop("OK\n")
}

func setIndexPageSize(db letarette.Database, pageSize int) {
	fmt.Printf("Setting page size to %v...\n", pageSize)
	err := letarette.SetIndexPageSize(db, pageSize)
	if err != nil {
		logger.Error.Printf("Failed to set page size: %v", err)
		return
	}
	fmt.Println("OK")
}

const statsTemplate = `
Index contains {{.Docs}} documents and {{.UniqueTerms}} unique terms of {{.TotalTerms}} in total.

Settings:
========
Languages: {{join .Stemmer.Stemmers ","}}
Token characters: {{printf "%q" .Stemmer.TokenCharacters}}
Separators: {{printf "%q" .Stemmer.Separators}}
Remove diacritics: {{if .Stemmer.RemoveDiacritics}}yes{{else}}no{{end}}

Spaces:
======
{{range .Spaces -}}
{{printf "☆ %s\t" .Name}} - Last updated @ {{nanoDate .State.LastUpdated}} ({{.State.LastUpdatedDocID}})
{{else}}No spaces
{{end}}
Top terms:
=========
{{range .CommonTerms -}}
{{printf "☆ %s\t%12d" .Term .Count}}
{{end}}
`

func printIndexStats(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Crunching numbers ")
	defer s.Stop()

	var err error
	stats, err := letarette.GetIndexStats(db)
	if err != nil {
		logger.Error.Printf("Failed to print index stats: %v", err)
		return
	}

	tmpl := template.New("stats")
	tmpl = tmpl.Funcs(template.FuncMap{
		"join": strings.Join,
		"nanoDate": func(nanos int64) string {
			return time.Unix(0, nanos).Format(time.RFC1123)
		},
	})
	tmpl, err = tmpl.Parse(statsTemplate)
	if err != nil {
		logger.Error.Printf("Failed to parse template: %v", err)
		return
	}

	s.Stop()
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 0, ' ', 0)
	err = tmpl.Execute(writer, &stats)
	if err != nil {
		logger.Error.Printf("Failed to execute template: %v", err)
	}
}

func optimizeIndex(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Optimizing index ")
	optimizer, err := letarette.StartIndexOptimization(db, 100)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to start optimizer: %v\n", err))
		return
	}
	defer optimizer.Close()
	for {
		done, err := optimizer.Step()
		if err != nil {
			s.Stop(fmt.Sprintf("Failed to run optimize step: %v\n", err))
			return
		}
		if done {
			break
		}
	}
	err = optimizer.Close()
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to close optimizer: %v\n", err))
		return
	}
	err = letarette.VacuumIndex(db)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to vacuum after optimize: %v\n", err))
		return
	}
	s.Stop("OK\n")
}

func rebuildIndex(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Rebuilding index ")

	err := letarette.RebuildIndex(db)
	if err == nil {
		err = letarette.VacuumIndex(db)
	}
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to rebuild index: %v\n", err))
		return
	}
	s.Stop("OK\n")
}

func compressIndex(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Compressing index ")

	ctx := context.Background()
	err := letarette.CompressIndex(ctx, db)
	if err == nil {
		err = letarette.VacuumIndex(db)
	}
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to compress index: %v\n", err))
		return
	}
	s.Stop("OK\n")
}

func forceIndexStemmerState(state snowball.Settings, db letarette.Database) {
	fmt.Println("Forcing stemmer state change...")
	err := letarette.ForceIndexStemmerState(state, db)
	if err != nil {
		logger.Error.Printf("Failed to force index update: %v", err)
		return
	}
	fmt.Println("OK")
}

func doMonitor(cfg letarette.Config) {
	fmt.Printf("Listening to status broadcasts...\n")
	listener := func(status protocol.IndexStatus) {
		logger.Info.Printf("%v\n", status)
	}
	m, err := client.NewMonitor(
		cfg.Nats.URLS,
		listener,
		client.WithSeedFile(cfg.Nats.SeedFile),
		client.WithRootCAs(cfg.Nats.RootCAs...),
	)
	if err != nil {
		logger.Error.Printf("Failed to create monitor: %v", err)
		return
	}
	defer m.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	<-signals
}

func resetMigration(cfg letarette.Config, version int) {
	fmt.Printf("Resetting migration to version %v...\n", version)
	err := letarette.ResetMigration(cfg, version)
	if err != nil {
		logger.Error.Printf("Failed to reset migration: %v", err)
		return
	}
	fmt.Println("OK")
}

func doSQL(cfg letarette.Config) {
	db, err := letarette.OpenDatabase(cfg)
	defer db.Close()

	if err != nil {
		logger.Error.Printf("Failed to open db: %v", err)
		return
	}

	statement := strings.Join(cmdline.Statement, " ")
	if strings.HasPrefix(statement, "@") {
		bytes, err := ioutil.ReadFile(strings.TrimLeft(statement, "@"))
		if err != nil {
			logger.Error.Printf("Failed to load statement file: %v", err)
			return
		}
		statement = string(bytes)
	}
	sql(db, statement)
}

func sql(db letarette.Database, statement string) {
	start := time.Now()
	result, err := db.RawQuery(statement)
	if err != nil {
		logger.Error.Printf("Failed to execute query: %v", err)
		return
	}
	duration := float32(time.Since(start)) / float32(time.Second)
	fmt.Printf("Executed in %vs\n", duration)
	for _, v := range result {
		fmt.Println(v)
	}
}

func updateSpelling(cfg letarette.Config) {
	s := spinner.New(os.Stdout)
	s.Start("Updating spelling ")

	db, err := letarette.OpenDatabase(cfg)
	defer db.Close()

	if err != nil {
		s.Stop(fmt.Sprintf("Failed to open db: %v", err))
		return
	}

	ctx := context.Background()
	err = letarette.UpdateSpellfix(ctx, db, cmdline.SpellingLimit)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to update spelling: %v", err))
		return
	}
	s.Stop("OK\n")
}
