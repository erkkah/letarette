package letarette

import (
	"github.com/nats-io/go-nats"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Version int `toml:"version"`
	Nats    struct {
		URL   string
		Topic string
	}
	Db struct {
		Path string
	}
	Index struct {
		Spaces []string
	}
}

func LoadConfig(configFile string) (cfg Config, err error) {
	cfg.Nats.URL = nats.DefaultURL
	cfg.Nats.Topic = "leta"
	cfg.Db.Path = "letarette.db"

	tree, err := toml.LoadFile(configFile)
	if err != nil {
		return
	}
	err = tree.Unmarshal(&cfg)
	// ??? Validate space names!
	return
}
