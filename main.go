package main

/*
	Letarette main application, the "worker".
	Communicates via "nats" message bus, maintains an index and responds to queries.
*/

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nats-io/go-nats"
	"github.com/pelletier/go-toml"
)

func createOrDie(db *sql.DB, sql string) {
	_, err := db.Exec(sql)
	if err != nil {
		log.Panicln("Failed to create table")
	}
}

func initDB(db *sql.DB, spaces []string) {
	createMeta := `create table if not exists meta (version, updated)`
	createOrDie(db, createMeta)

	createStatus := `create table if not exists status (space, lastUpdate)`
	createOrDie(db, createStatus)

	createInterest := `create table if not exists interest (space, docID)`
	createOrDie(db, createInterest)

	for _, space := range spaces {
		createIndex := fmt.Sprintf(
			`create virtual table if not exists %q using fts5(`+
				`txt, updated unindexed, hash unindexed, `+
				`tokenize="porter unicode61 tokenchars '#'");`, space)
		createOrDie(db, createIndex)
	}
}

type config struct {
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

func loadConfig(configFile string) (cfg config, err error) {
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

func parseAndExecute(q string) string {
	return q + ":???"
}

func main() {
	load := flag.String("load", "", "Load entries from file, line by line")
	match := flag.Bool("match", false, "Read from stdin and match each line")

	conf := flag.String("conf", "letarette.toml", "Configuration TOML file")
	flag.Parse()

	cfg, err := loadConfig(*conf)
	if err != nil {
		log.Panic("Failed to load config:", err)
	}

	log.Printf("Connecting to nats server at %q\n", cfg.Nats.URL)
	conn, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		log.Panicf("Failed to connect to nats server")
	}
	defer conn.Close()

	db, err := sql.Open("sqlite3", cfg.Db.Path)
	if err != nil {
		panic(err)
	}

	spaces := cfg.Index.Spaces
	log.Printf("%v\n", cfg)
	if len(spaces) < 1 {
		log.Panicf("No spaces defined: %v", spaces)
	}
	initDB(db, spaces)

	conn.Subscribe(cfg.Nats.Topic+".q", func(m *nats.Msg) {
		// Handle query
		reply := parseAndExecute(string(m.Data))
		// Reply
		conn.Publish(m.Reply, []byte(reply))
	})

	if *load != "" {
		wordFile := os.Stdin
		if *load != "-" {
			wordFile, err = os.Open(*load)
			if err != nil {
				panic(err)
			}
		}
		tx, err := db.Begin()
		if err != nil {
			panic(err)
		}
		st, err := tx.Prepare("insert into stuff (txt) values(?)")
		if err != nil {
			panic(err)
		}
		fileScanner := bufio.NewScanner(wordFile)
		loaded := 0
		for fileScanner.Scan() {
			line := fileScanner.Text()
			line = strings.TrimSpace(line)
			if line != "" {
				_, err = st.Exec(line)
				if err != nil {
					panic(err)
				}
				loaded++
			}
		}
		err = tx.Commit()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Loaded %v items\n", loaded)
	}

	if *match {
		rows, err := db.Query("select count(*) from stuff")
		if err != nil {
			panic(err)
		}
		var phrases int32
		rows.Next()
		err = rows.Scan(&phrases)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v phrases loaded\n", phrases)

		st, err := db.Prepare("select rowid, txt, rank from stuff where txt match ? order by rank limit 10")
		if err != nil {
			panic(err)
		}

		matchScanner := bufio.NewScanner(os.Stdin)
		for matchScanner.Scan() {
			line := matchScanner.Text()
			start := time.Now()
			rows, err := st.Query(line)
			t1 := time.Now()
			if err != nil {
				panic(err)
			}
			hits := 0
			for rows.Next() {
				var rowid int64
				var hit string
				var rank float32
				if err := rows.Scan(&rowid, &hit, &rank); err != nil {
					panic(err)
				}
				hits++
				if len(hit) > 64 {
					hit = hit[:64] + "..."
				}
				fmt.Printf("%v: %q (%v)\n", rowid, hit, rank)
			}
			rows.Close()
			t2 := time.Now()
			dur1 := t1.Sub(start)
			dur2 := t2.Sub(t1)

			fmt.Printf("--- %v hits in (%v + %v) ---\n", hits, dur1, dur2)
		}
	}
	_ = db.Close()
}
