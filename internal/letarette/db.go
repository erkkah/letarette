package letarette

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Load SQLite driver
)

type Database interface {
	Close()
	GetRawDB() *sql.DB

	getLastUpdateTime(string) (time.Time, error)
	setLastUpdateTime(string, time.Time) error

	setInterestList(string, []string) error
	getInterestList(string) ([]string, error)
}

type database struct {
	db *sqlx.DB
}

func OpenDatabase(cfg Config) (Database, error) {
	db, err := openDatabase(cfg)
	if err != nil {
		return nil, err
	}

	newDB := &database{db}
	return newDB, nil
}

func (db *database) Close() {
	db.db.Close()
}

func (db *database) getLastUpdateTime(space string) (t time.Time, err error) {
	rows, err := db.db.Query("select lastUpdate from spaces where space = ?", space)
	if err != nil {
		return
	}

	if rows.Next() {
		err = rows.Scan(&t)
	} else {
		err = fmt.Errorf("Cannot get last update time for space %q", space)
	}
	return
}

func (db *database) setLastUpdateTime(space string, t time.Time) error {
	res, err := db.db.Exec("update spaces set lastUpdate = ? where space = ?", t, space)
	if err != nil {
		return err
	}

	count, _ := res.RowsAffected()
	if count != 1 {
		err = fmt.Errorf("Failed to set update time for space %q", space)
	}

	return err
}

func (db *database) getInterestList(space string) (result []string, err error) {
	rows, err := db.db.Query(`
		select docID from interest
		left join spaces on interest.spaceID = spaces.spaceID
		where space = ?`, space)
	if err != nil {
		return
	}
	for rows.Next() {
		var docID string
		err = rows.Scan(&docID)
		if err != nil {
			return
		}
		result = append(result, docID)
	}
	return
}

func (db *database) setInterestList(space string, list []string) error {
	tx, err := db.db.Beginx()
	var spaceID int
	err = tx.Get(&spaceID, `select spaceID from spaces where name = ?`, space)
	if err != nil {
		return err
	}
	var interestCount int
	err = tx.Get(&interestCount, `select count(*) from interests where spaceID = ?`, spaceID)
	if err != nil {
		return err
	}
	if interestCount != 0 {
		return fmt.Errorf("Cannot overwrite current interest list")
	}
	st, err := tx.Preparex(`insert into interest (spaceID, docID), values(?, ?)`)
	if err != nil {
		return err
	}
	for docID := range list {
		_, err := st.Exec(spaceID, docID)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	return err
}

func (db *database) GetRawDB() *sql.DB {
	return db.db.DB
}

func createOrDie(db *sqlx.DB, sql string) {
	_, err := db.Exec(sql)
	if err != nil {
		log.Panicf("Failed to create table: %v", err)
	}
}

func initDB(db *sqlx.DB, spaces []string) {
	createMeta := `create table if not exists meta (version, updated)`
	createOrDie(db, createMeta)

	createSpaces := `create table if not exists spaces(
		spaceID integer primary key,
		space text not null unique,
		lastUpdate datetime not null,
		listCreatedAt datetime
	)`
	createOrDie(db, createSpaces)

	createInterest := `create table if not exists interest(
		spaceID not null,
		docID text not null,
		unique(spaceID, docID)
		foreign key (spaceID) references spaces(spaceID)
	)`
	createOrDie(db, createInterest)

	for _, space := range spaces {
		indexTable := fmt.Sprintf(`space_%v`, space)
		createIndex := fmt.Sprintf(
			`create virtual table if not exists %q using fts5(
				txt, updated unindexed, hash unindexed,
				tokenize="porter unicode61 tokenchars '#'"
			);`, indexTable)
		createOrDie(db, createIndex)

		createSpace := `insert into spaces (space, lastUpdate) values(?, 0)`
		_, err := db.Exec(createSpace, space)
		if err != nil {
			log.Panicf("Failed to create space table: %v", err)
		}
	}
}

func openDatabase(cfg Config) (db *sqlx.DB, err error) {
	db, err = sqlx.Connect("sqlite3", cfg.Db.Path)
	if err != nil {
		return
	}
	spaces := cfg.Index.Spaces
	if len(spaces) < 1 {
		log.Panicf("No spaces defined: %v", spaces)
	}
	initDB(db, spaces)
	return
}
