package letarette

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URL         string `default:"nats://localhost:4222"`
		Topic       string `default:"leta"`
		SearchGroup string `default:"1"`
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
	Stemmer struct {
		Languages        []string `split_words:"true" default:"english"`
		RemoveDiacritics bool     `split_words:"true" default:"true"`
		TokenCharacters  string
		Separators       string
	}
	Search struct {
		Timeout      time.Duration `default:"200ms"`
		CacheTimeout time.Duration `split_words:"true" default:"1m"`
	}
	MetricsPort uint16 `split_words:"true" default:"8000"`
}

const prefix = "LETARETTE"

// LoadConfig loads configuration variables from the environment
// and returns a fully populated Config instance.
func LoadConfig() (cfg Config, err error) {
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

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	envconfig.Usage(prefix, &cfg)
}
