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
}

func main() {
	usage := `Letarette CLI.

Usage:
    lrcli [-n <url>] [-v] <space> <phrase>...

Options:
	-n <url>    NATS url to connect to [default: nats://localhost:4222]
	-v          Verbose
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

	res, err := c.Search(strings.Join(config.Phrases, " "), []string{config.Space}, 10)
	if err != nil {
		log.Panicf("Failed to perform search")
	}

	for _, doc := range res.Documents {
		fmt.Println(doc.Snippet)
	}
}
