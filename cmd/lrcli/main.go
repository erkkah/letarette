package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/pkg/client"
)

var config struct {
	Space   string   `docopt:"<space>"`
	NatsURL string   `docopt:"-n"`
	Verbose bool     `docopt:"-v"`
	Phrases []string `docopt:"<phrase>"`
	Limit   int      `docopt:"-l"`
	Offset  int      `docopt:"-o"`
}

func main() {
	usage := `Letarette CLI.

Usage:
    lrcli [-n <url>] [-v] [-l <limit>] [-o <offset>] <space> <phrase>...

Options:
    -n <url>     NATS url to connect to [default: nats://localhost:4222]
    -v           Verbose
    -l <limit>   Limit [default: 10]
    -o <offset>  Offset [default: 0]
`

	args, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Panicf("Failed to parse args: %v", err)
	}
	err = args.Bind(&config)
	if err != nil {
		log.Panicf("Failed to bind args: %v", err)
	}

	c, err := client.NewSearchClient(config.NatsURL)
	if err != nil {
		log.Panicf("Failed to create search client: %v", err)
	}
	defer c.Close()

	res, err := c.Search(strings.Join(config.Phrases, " "), []string{config.Space}, config.Limit, config.Offset)
	if err != nil {
		log.Panicf("Failed to perform search: %v", err)
	}

	fmt.Printf("Query executed in %v seconds with status %q\n\n", res.Duration, res.Status.String())
	for _, doc := range res.Documents {
		fmt.Println(doc.Snippet)
	}
}
