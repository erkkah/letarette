package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/erkkah/letarette/pkg/charma"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"

	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
)

var cmdline struct {
	Space   string   `docopt:"<space>"`
	Verbose bool     `docopt:"-v"`
	Phrases []string `docopt:"<phrase>"`
	Limit   int      `docopt:"-l"`
	Offset  int      `docopt:"-o"`

	Search       bool
	Index        bool
	Stats        bool
	Check        bool
	Forcestemmer bool
	Rebuild      bool
	Env          bool
}

func main() {
	title := charma.CircleCode("LETARETTE")
	usage := title + `

Usage:
	lrcli search [-v] [-l <limit>] [-o <offset>] <space> <phrase>...
	lrcli index stats
	lrcli index check
	lrcli index rebuild
	lrcli index forcestemmer
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
		if err != nil {
			logger.Error.Printf("Failed to open db: %v", err)
			return
		}
		switch {
		case cmdline.Check:
			checkIndex(db)
		case cmdline.Stats:
			printIndexStats(db)
		case cmdline.Rebuild:
			rebuildIndex(db)
		case cmdline.Forcestemmer:
			settings := snowball.Settings{
				Stemmers:         cfg.Stemmer.Languages,
				RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
				Separators:       cfg.Stemmer.Separators,
				TokenCharacters:  cfg.Stemmer.TokenCharacters,
			}
			forceIndexStemmerState(settings, db)
		}
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

const statsTemplate = `
Index contains {{.Docs}} documents and {{.Terms}} unique terms.

{{"Settings:" | underline}}
Languages: {{join .Stemmer.Stemmers ","}}
Token characters: {{printf "%q" .Stemmer.TokenCharacters}}
Separators: {{printf "%q" .Stemmer.Separators}}
Remove diacritics: {{if .Stemmer.RemoveDiacritics}}yes{{else}}no{{end}}

{{"Spaces:" | underline}}
{{range .Spaces -}}
{{printf "\u23f5%s\t" .Name}} - Last updated @ {{nanoDate .State.LastUpdated}} ({{.State.LastUpdatedDocID}})
{{else}}No spaces
{{end}}
{{"Top terms:" | underline}}
{{range .CommonTerms -}}
{{printf "\u23f5%s\t%12d" .Term .Count}}
{{end}}
`

func printIndexStats(db letarette.Database) {
	fmt.Println("Crunching numbers...")

	/*
		stats := letarette.Stats{
			Spaces: []struct {
				Name  string
				State letarette.InterestListState
			}{
				{"spejset", letarette.InterestListState{
					CreatedAt:        10,
					LastUpdated:      20,
					LastUpdatedDocID: "ABABA",
				}},
				{"rymden", letarette.InterestListState{
					CreatedAt:        30,
					LastUpdated:      40,
					LastUpdatedDocID: "BEBEB",
				}},
			},
			CommonTerms: []struct {
				Term  string
				Count int
			}{
				{"korv", 22},
				{"apa", 21},
			},
			Terms: 11,
			Docs:  4,
		}
	*/

	var err error
	stats, err := letarette.GetIndexStats(db)
	if err != nil {
		logger.Error.Printf("Failed to print index stats: %v", err)
		return
	}

	tmpl := template.New("stats")
	wordChar, _ := regexp.Compile(`\w`)
	tmpl = tmpl.Funcs(template.FuncMap{
		"join": strings.Join,
		"nanoDate": func(nanos int64) string {
			return time.Unix(0, nanos).Format(time.RFC1123)
		},
		"underline": func(str string) string {
			return wordChar.ReplaceAllString(str, "$0\u20e8")
		},
	})
	tmpl, err = tmpl.Parse(statsTemplate)
	if err != nil {
		logger.Error.Printf("Failed to parse template: %v", err)
		return
	}

	tmpl.Execute(os.Stdout, &stats)
}

func rebuildIndex(db letarette.Database) {
	fmt.Println("Rebuilding index...")
	err := letarette.RebuildIndex(db)
	if err != nil {
		logger.Error.Printf("Failed to rebuild index: %v", err)
		return
	}
	fmt.Println("OK")
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
		fmt.Println(doc.Snippet)
	}
}
