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
	c, err := client.NewSearchClient("nats://localhost:4222")
	if err != nil {
		fmt.Printf("NATS connection failed: %v", err)
		return
	}
	defer c.Close()

	spaces := []string{"fruits"}
	limit := 10
	offset := 0

	res, err := c.Search("apple", spaces, limit, offset)
	if err != nil {
		fmt.Printf("Search request failed: %v", err)
		return
	}

	for _, hit := range res.Result.Hits {
		fmt.Println(hit.Snippet)
	}
}


```