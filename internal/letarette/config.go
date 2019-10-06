package letarette

import (
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
}

func LoadConfig(configFile string) (cfg Config, err error) {
	prefix := "LETARETTE"
	err = envconfig.Process(prefix, &cfg)

	if err != nil {
		envconfig.Usage(prefix, &cfg)
	}

	// ??? Validate space names!
	// ??? Validate chunk size!
	// ??? Validate wait time!
	return
}
