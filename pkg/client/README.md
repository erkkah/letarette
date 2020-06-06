[![GoDoc](https://godoc.org/github.com/erkkah/letarette/pkg/client?status.svg)](https://godoc.org/github.com/erkkah/letarette/pkg/client)

## Letarette client library for Golang

[Letarette docs](https://letarette.io/docs)

### Installation

```
go get -u github.com/erkkah/letarette
```

### Example

```go
package main

import (
	"fmt"

	"github.com/erkkah/letarette/pkg/client"
)

func main() {
	agent, err := client.NewSearchAgent([]string{"nats://localhost:4222"})
	if err != nil {
		fmt.Printf("NATS connection failed: %v", err)
		return
	}
	defer agent.Close()

	spaces := []string{"fruits"}
	pageLimit := 10
	pageOffset := 0

	res, err := agent.Search("apple", spaces, pageLimit, pageOffset)
	if err != nil {
		fmt.Printf("Search request failed: %v", err)
		return
	}

	for _, hit := range res.Result.Hits {
		fmt.Println(hit.Snippet)
	}
}


```