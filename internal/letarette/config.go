package letarette

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URL   string `default:"nats://localhost:4222"`
		Topic string `default:"leta"`
	}
	Db struct {
		Path string `default:"letarette.db"`
	}
	Index struct {
		Spaces          []string      `required:"true"`
		ChunkSize       uint16        `default:"100"`
		MaxInterestWait time.Duration `default:"5s"`
		MaxDocumentWait time.Duration `default:"1s"`
		CycleWait       time.Duration `default:"100ms"`
		MaxOutstanding  uint16        `default:"10"`
	}
	MetricsPort uint16 `split_words:"true" default:"8000"`
}

// LoadConfig loads configuration variables from the environment
// and returns a fully populated Config instance.
func LoadConfig() (cfg Config, err error) {
	prefix := "LETARETTE"
	err = envconfig.Process(prefix, &cfg)

	if err != nil {
		envconfig.Usage(prefix, &cfg)
	}

	unique := map[string]string{}
	for _, v := range cfg.Index.Spaces {
		unique[v] = v
	}
	if len(unique) != len(cfg.Index.Spaces) {
		return Config{}, fmt.Errorf("Space names must be unique")
	}

	return
}
