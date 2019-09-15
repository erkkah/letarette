package letarette

//go:generate go-bindata -pkg $GOPACKAGE -o migrations.go migrations/

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Load SQLite driver

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" // Load SQLite migration driver
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

type Interest struct {
	DocID  DocumentID `db:"docID"`
	Served bool
}

type InterestListState struct {
	CreatedAt   time.Time `db:"listCreatedAt"`
	UpdateStart time.Time `db:"listUpdateStart"`
	UpdateEnd   time.Time `db:"listUpdateEnd"`
	ChunkStart  uint64    `db:"chunkStart"`
	ChunkSize   uint16    `db:"chunkSize"`
}

// Database is a live connection to a SQLite database file,
// providing access methods for all db interactions.
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

// OpenDatabase connects to a new or existing database and
// migrates the database up to the latest version.
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
		update spaces set
			listCreatedAt = datetime('now'),
			listUpdateStart = 0,
			listUpdateEnd = 0,
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

func initDB(db *sqlx.DB, sqliteURL string, spaces []string) error {
	migrations, err := AssetDir("migrations")
	if err != nil {
		return err
	}
	res := bindata.Resource(migrations, func(name string) ([]byte, error) {
		return Asset("migrations/" + name)
	})

	driver, err := bindata.WithInstance(res)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", driver, "sqlite3://"+sqliteURL)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	for _, space := range spaces {

		createSpace := `insert into spaces (space, lastUpdate, chunkStart, chunkSize) values(?, 0, 0, 0) on conflict do nothing`
		_, err := db.Exec(createSpace, space)
		if err != nil {
			return fmt.Errorf("Failed to create space table: %w", err)
		}
	}

	return nil
}

func openDatabase(cfg Config) (db *sqlx.DB, err error) {
	abspath, err := filepath.Abs(cfg.Db.Path)
	if err != nil {
		return nil, fmt.Errorf("Failed to get absolute path to DB: %w", err)
	}
	escapedPath := strings.Replace(abspath, " ", "%20", -1)
	sqliteURL := fmt.Sprintf("file:%s?_journal=WAL", escapedPath)

	db, err = sqlx.Connect("sqlite3", sqliteURL)
	if err != nil {
		return
	}
	spaces := cfg.Index.Spaces
	if len(spaces) < 1 {
		return nil, fmt.Errorf("No spaces defined: %v", spaces)
	}
	err = initDB(db, sqliteURL, spaces)
	return
}
