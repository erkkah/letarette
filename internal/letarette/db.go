package letarette

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Load SQLite driver
)

type Interest struct {
	DocID  string `db:"docID"`
	Served bool
}

type InterestListState struct {
	CreatedAt   time.Time `db:"listCreatedAt"`
	UpdateStart time.Time `db:"listUpdateStart"`
	UpdateEnd   time.Time `db:"listUpdateEnd"`
	ChunkStart  uint64    `db:"chunkStart"`
	ChunkSize   uint16    `db:"chunkSize"`
}

type Database interface {
	Close()
	GetRawDB() *sql.DB

	getLastUpdateTime(string) (time.Time, error)
	setLastUpdateTime(string, time.Time) error

	setInterestList(string, []DocumentID) error
	getInterestList(string) ([]Interest, error)

	getInterestListState(string) (InterestListState, error)
	setChunkStart(space string, start uint64) error
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
	err = db.db.Get(&t, "select lastUpdate from spaces where space = ?", space)
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

func (db *database) getInterestListState(space string) (state InterestListState, err error) {
	err = db.db.Get(&state,
		`select listCreatedAt, listUpdatedStart, listUpdateEnd, chunkStart from spaces where space = ?`, space)
	return
}

func (db *database) setChunkStart(space string, start uint64) error {
	_, err := db.db.Exec("update spaces set chunkStart = ? where space = ?", start, space)
	return err
}

func (db *database) getInterestList(space string) (result []Interest, err error) {
	var spaceID int
	err = db.db.Get(&spaceID, `select spaceID from spaces where space = ?`, space)
	if err != nil {
		return
	}
	rows, err := db.db.Queryx(`
		select docID, served from interest
		where spaceID = ?`, spaceID)
	if err != nil {
		return
	}
	for rows.Next() {
		var interest Interest
		err = rows.StructScan(&interest)
		if err != nil {
			return
		}
		result = append(result, interest)
	}
	return
}

func (db *database) setInterestList(space string, list []DocumentID) error {
	tx, err := db.db.Beginx()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var spaceID int
	err = tx.Get(&spaceID, `select spaceID from spaces where space = ?`, space)
	if err != nil {
		return err
	}
	var interestCount int
	err = tx.Get(&interestCount, `select count(*) from interest where spaceID = ?`, spaceID)
	if err != nil {
		return err
	}
	if interestCount != 0 {
		return fmt.Errorf("Cannot overwrite current interest list")
	}
	st, err := tx.Preparex(`insert into interest (spaceID, docID, served) values(?, ?, 0)`)
	if err != nil {
		return err
	}
	for _, docID := range list {
		_, err := st.Exec(spaceID, docID)
		if err != nil {
			return err
		}
	}
	_, err = tx.NamedExec(`
		with now as (select datetime('now'))
		update spaces set
			listCreatedAt = now,
			listUpdateStart = 0,
			listUpdateEnd = 0
			chunkSize = :chunkSize
		where spaceID = :spaceID`,

		map[string]interface{}{
			"spaceID":   spaceID,
			"chunkSize": len(list),
		})
	if err != nil {
		return err
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

		-- timestamp of where we are in the index update process
		lastUpdate datetime not null,
		-- interest list creation timestamp
		listCreatedAt datetime,
		-- timestamp when the newest list entry was updated
		listUpdatedAt datetime,
		-- offset into documents starting with the same timestamp
		chunkStart integer not null

		check(
			listUpdatedAt <= listCreatedAt and
			lastUpdate <= listCreatedAt
		)
	)`
	createOrDie(db, createSpaces)

	createInterest := `create table if not exists interest(
		spaceID integer not null,
		docID text not null,
		served integer not null,
		unique(spaceID, docID)
		foreign key (spaceID) references spaces(spaceID)
	)`
	createOrDie(db, createInterest)

	for _, space := range spaces {
		indexTable := fmt.Sprintf(`space_%v`, space)
		createIndex := fmt.Sprintf(
			`create virtual table if not exists %q using fts5(
				txt, updated unindexed, docID unindexed,
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
