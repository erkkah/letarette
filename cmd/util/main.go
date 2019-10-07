package main

/*
	Letarette utility application
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

	"github.com/erkkah/letarette/internal/letarette"
)

func loadText(input string, db *sql.DB) {
	wordFile := os.Stdin
	var err error
	if input != "-" {
		wordFile, err = os.Open(input)
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

func matchText(db *sql.DB) {
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

func main() {
	load := flag.String("load", "", "Load entries from file, line by line")
	match := flag.Bool("match", false, "Read from stdin and match each line")

	cfg, err := letarette.LoadConfig()
	if err != nil {
		log.Panic("Failed to load config:", err)
	}

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if *load != "" {
		loadText(*load, db.GetRawDB())
	}

	if *match {
		matchText(db.GetRawDB())
	}

}
