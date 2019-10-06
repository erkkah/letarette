package letarette

import (
	"github.com/nats-io/go-nats"

	"github.com/erkkah/letarette/pkg/logger"
)

type Searcher interface {
	Close()
}

type searcher struct {
	closer chan bool
}

func (s *searcher) Close() {
	assert(s.closer != nil, "Searcher close channel is not nil")
	s.closer <- true
	<-s.closer
}

func parseAndExecute(q string) string {
	return "whqhqw"
}

func StartSearcher(nc *nats.Conn, db Database, cfg Config) Searcher {
	closer := make(chan bool, 0)

	nc.Subscribe(cfg.Nats.Topic+".q", func(m *nats.Msg) {
		// Handle query
		reply := parseAndExecute(string(m.Data))
		// Reply
		nc.Publish(m.Reply, []byte(reply))
	})

	go func() {
		// for ever:
		logger.Info.Printf("Searcher starting")
		<-closer
		logger.Info.Printf("Searcher exiting")
		closer <- true
	}()

	return &searcher{closer}
}
