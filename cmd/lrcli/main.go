package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/briandowns/spinner"
	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"

	"github.com/erkkah/letarette/pkg/charma"
	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
)

var cmdline struct {
	Verbose bool `docopt:"-v"`

	Search  bool
	Space   string   `docopt:"<space>"`
	Phrases []string `docopt:"<phrase>"`
	Limit   int      `docopt:"-l"`
	Offset  int      `docopt:"-o"`

	Index        bool
	Stats        bool
	Check        bool
	Pgsize       bool
	Size         int `docopt:"<size>"`
	Rebuild      bool
	Optimize     bool
	ForceStemmer bool `docopt:"forcestemmer"`

	SQL       bool     `docopt:"sql"`
	Statement []string `docopt:"<sql>"`

	ResetMigration bool `docopt:"resetmigration"`
	Version        int  `docopt:"<version>"`

	Env bool
}

func main() {
	title := charma.CircleChars("Letarette")
	usage := title + `

Usage:
	lrcli search [-v] [-l <limit>] [-o <offset>] <space> <phrase>...
	lrcli sql <sql>...
	lrcli index stats
	lrcli index check
	lrcli index pgsize <size>
	lrcli index optimize
	lrcli index rebuild
	lrcli index forcestemmer
	lrcli resetmigration <version>
	lrcli env

Options:
    -v           Verbose
    -l <limit>   Search result limit [default: 10]
    -o <offset>  Search result offset [default: 0]
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

	if cmdline.Env {
		letarette.Usage()
	} else if cmdline.Search {
		doSearch(cfg)
	} else if cmdline.Index {
		db, err := letarette.OpenDatabase(cfg)
		defer func() {
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
		case cmdline.Check:
			err = letarette.CheckStemmerSettings(db, cfg)
			if err == letarette.ErrStemmerSettingsMismatch {
				logger.Warning.Printf("Index and config stemmer settings mismatch. Re-build index or force changes.")
			}
			checkIndex(db)
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
	} else if cmdline.ResetMigration {
		resetMigration(cfg, cmdline.Version)
	} else if cmdline.SQL {
		db, err := letarette.OpenDatabase(cfg)
		defer db.Close()

		if err != nil {
			logger.Error.Printf("Failed to open db: %v", err)
			return
		}

		sql(db, strings.Join(cmdline.Statement, " "))
	}
}

func checkIndex(db letarette.Database) {
	fmt.Println("Checking index...")
	err := letarette.CheckIndex(db)
	if err != nil {
		logger.Error.Printf("Index check failed: %v", err)
		return
	}
	fmt.Println("OK")
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
Index contains {{.Docs}} documents and {{.Terms}} unique terms.

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
	s := getSpinner("Crunching numbers ", "\n")
	s.Start()
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
	tmpl.Execute(os.Stdout, &stats)
}

func optimizeIndex(db letarette.Database) {
	s := getSpinner("Optimizing index ", "OK\n")
	s.Start()
	defer s.Stop()
	optimizer, err := letarette.StartIndexOptimization(db, 100)
	if err != nil {
		logger.Error.Printf("Failed to start optimizer: %w", err)
		return
	}
	defer optimizer.Close()
	for {
		done, err := optimizer.Step()
		if err != nil {
			logger.Error.Printf("Failed to run optimize step: %w", err)
			return
		}
		if done {
			break
		}
	}
}

func rebuildIndex(db letarette.Database) {
	s := getSpinner("Rebuilding index ", "OK\n")
	s.Start()
	defer s.Stop()

	err := letarette.RebuildIndex(db)
	if err != nil {
		logger.Error.Printf("Failed to rebuild index: %v", err)
		return
	}
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

func doSearch(cfg letarette.Config) {
	c, err := client.NewSearchClient(cfg.Nats.URL)
	if err != nil {
		logger.Error.Printf("Failed to create search client: %v", err)
		return
	}
	defer c.Close()

	res, err := c.Search(strings.Join(cmdline.Phrases, " "), []string{cmdline.Space}, cmdline.Limit, cmdline.Offset)
	if err != nil {
		logger.Error.Printf("Failed to perform search: %v", err)
		return
	}

	fmt.Printf("Query executed in %v seconds with status %q\n\n", res.Duration, res.Status.String())
	for _, doc := range res.Documents {
		fmt.Printf("[%v] %s\n", doc.ID, doc.Snippet)
	}
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

func getSpinner(labels ...string) *spinner.Spinner {
	spnr := spinner.New(spinner.CharSets[2], time.Millisecond*500)
	spnr.Color("yellow", "bold")
	if len(labels) > 0 {
		spnr.Prefix = labels[0]
	}
	if len(labels) > 1 {
		spnr.FinalMSG = labels[1]
	}
	return spnr
}

func sql(db letarette.Database, statement string) {
	result, err := db.RawQuery(statement)
	if err != nil {
		logger.Error.Printf("Failed to execute query: %v", err)
		return
	}
	for _, v := range result {
		fmt.Println(v)
	}
}
