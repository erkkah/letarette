package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/briandowns/spinner"
	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"

	"github.com/erkkah/letarette/pkg/charma"
	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

var cmdline struct {
	Verbose bool `docopt:"-v"`

	Search      bool
	Space       string   `docopt:"<space>"`
	Phrases     []string `docopt:"<phrase>"`
	PageLimit   int      `docopt:"-l"`
	PageOffset  int      `docopt:"-p"`
	GroupSize   int32    `docopt:"-g"`
	Interactive bool     `docopt:"-i"`

	Monitor bool

	SQL       bool     `docopt:"sql"`
	Statement []string `docopt:"<sql>"`

	Index        bool
	Stats        bool
	Check        bool
	Pgsize       bool
	Size         int `docopt:"<size>"`
	Rebuild      bool
	Optimize     bool
	ForceStemmer bool `docopt:"forcestemmer"`

	ResetMigration bool `docopt:"resetmigration"`
	Version        int  `docopt:"<version>"`

	Env bool
}

func main() {
	title := charma.CircleChars("Letarette")
	usage := title + `

Usage:
    lrcli search [-v] [-l <limit>] [-p <page>] [-g <groupsize>] [-i] <space> [<phrase>...]
    lrcli monitor
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
    -v             Verbose
    -l <limit>     Search result page limit [default: 10]
    -p <page>      Search result page [default: 0]
    -i             Interactive search
    -g <groupsize> Force shard group size, do not discover
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
	cfg.Db.ToolConnection = true

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
	} else if cmdline.Monitor {
		doMonitor(cfg)
	}
}

func checkIndex(db letarette.Database) {
	s := getSpinner("Checking index ", "OK\n")
	s.Start()
	defer s.Stop()

	err := letarette.CheckIndex(db)
	if err != nil {
		logger.Error.Printf("Index check failed: %v", err)
		return
	}
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
	s := getSpinner("Crunching numbers ", "")
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
	c, err := client.NewSearchClient(cfg.Nats.URL, client.WithShardgroupSize(cmdline.GroupSize))
	if err != nil {
		logger.Error.Printf("Failed to create search client: %v", err)
		return
	}
	defer c.Close()

	os.Stdout.WriteString(">")
	if cmdline.Interactive {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			searchPhrase(scanner.Text(), c)
			os.Stdout.WriteString(">")
		}
	} else {
		searchPhrase(strings.Join(cmdline.Phrases, " "), c)
	}
}

func searchPhrase(phrase string, client client.SearchClient) {
	res, err := client.Search(
		phrase,
		[]string{cmdline.Space},
		cmdline.PageLimit,
		cmdline.PageOffset,
	)
	if err != nil {
		logger.Error.Printf("Failed to perform search: %v", err)
		return
	}

	fmt.Printf("Query executed in %v seconds with status %q\n", res.Duration, res.Status.String())
	fmt.Printf("Returning %v of %v total hits, capped: %v\n\n", len(res.Result.Hits), res.Result.TotalHits, res.Result.Capped)
	for _, doc := range res.Result.Hits {
		fmt.Printf("[%v] %s\n", doc.ID, doc.Snippet)
	}
}

func doMonitor(cfg letarette.Config) {
	fmt.Printf("Listening to status broadcasts...\n")
	listener := func(status protocol.IndexStatus) {
		fmt.Printf("%v\n", status)
	}
	m, err := client.NewMonitor(cfg.Nats.URL, listener)
	if err != nil {
		logger.Error.Printf("Failed to create monitor: %v", err)
	}
	defer m.Close()

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT)

	select {
	case <-signals:
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
