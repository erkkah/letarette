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

	"github.com/erkkah/letarette/pkg/protocol"
)

// InterestState represents the state of an interest
type InterestState int

const (
	pending InterestState = iota
	requested
	served
)

// Interest represents one row in the interest list
type Interest struct {
	DocID protocol.DocumentID `db:"docID"`
	State InterestState
}

// InterestListState keeps track of where the index process is
type InterestListState struct {
	CreatedAt        int64               `db:"listCreatedAtNanos"`
	LastUpdated      int64               `db:"lastUpdatedAtNanos"`
	LastUpdatedDocID protocol.DocumentID `db:"lastUpdatedDocID"`
}

func (state InterestListState) lastUpdatedTime() time.Time {
	return time.Unix(0, state.LastUpdated)
}

func (state InterestListState) createdAtTime() time.Time {
	return time.Unix(0, state.CreatedAt)
}

// Database is a live connection to a SQLite database file,
// providing access methods for all db interactions.
type Database interface {
	Close()
	GetRawDB() *sql.DB

	addDocumentUpdate(doc protocol.Document) error
	commitInterestList(space string) error
	getLastUpdateTime(string) (time.Time, error)

	clearInterestList(string) error
	resetRequested(string) error

	setInterestList(string, []protocol.DocumentID) error
	getInterestList(string) ([]Interest, error)
	setInterestState(string, protocol.DocumentID, InterestState) error

	getInterestListState(string) (InterestListState, error)
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
	var timestampNanos int64
	err = db.db.Get(&timestampNanos, "select lastUpdatedAtNanos from spaces where space = ?", space)
	t = time.Unix(0, timestampNanos)
	return
}

func (db *database) addDocumentUpdate(doc protocol.Document) error {
	spaceID, err := db.getSpaceID(doc.Space)
	if err != nil {
		return err
	}

	tx, err := db.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	log.Printf("Inserting doc from %v\n", doc.Updated.String())
	res, err := tx.Exec(`replace into docs (spaceID, docID, updatedNanos, txt) values (?, ?, ?, ?)`,
		spaceID, doc.ID, doc.Updated.UnixNano(), doc.Text)

	if err != nil {
		return fmt.Errorf("Failed to update doc: %w", err)
	}

	updatedRows, _ := res.RowsAffected()
	if updatedRows != 1 {
		return fmt.Errorf("Failed to update index, no rows affected")
	}

	_, err = tx.Exec(`update interest set state=? where spaceID=? and docID=?`, served, spaceID, doc.ID)

	if err != nil {
		return fmt.Errorf("Failed to update interest list: %w", err)
	}

	err = tx.Commit()
	if err == nil {
		tx = nil
	}

	return err
}

func (db *database) commitInterestList(space string) error {
	tx, err := db.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	var indexPosition struct {
		Updated int64               `db:"updatedNanos"`
		DocID   protocol.DocumentID `db:"docID"`
	}

	err = tx.Get(&indexPosition, `
		with listState as (
			select listCreatedAtNanos from spaces where space = ?
		)
		select docs.updatedNanos, docs.docID from interest left join docs using(docID)
		cross join listState
		where interest.state = ? and docs.updatedNanos < listState.listCreatedAtNanos
		order by docs.updatedNanos desc, docs.docID
		limit 1;`, space, served)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	res, err := tx.Exec("update spaces set lastUpdatedAtNanos = ?, lastUpdatedDocID = ? where space = ?",
		indexPosition.Updated, indexPosition.DocID, space)
	if err != nil {
		return err
	}

	count, _ := res.RowsAffected()
	if count != 1 {
		err = fmt.Errorf("Failed to update index position for space %q", space)
		return err
	}

	err = tx.Commit()
	if err == nil {
		tx = nil
	}

	return err
}

func (db *database) getInterestListState(space string) (state InterestListState, err error) {
	err = db.db.Get(&state,
		`select listCreatedAtNanos, lastUpdatedAtNanos, lastUpdatedDocID from spaces where space = ?`, space)
	return
}

func (db *database) getSpaceID(space string) (int, error) {
	var spaceID int
	err := db.db.Get(&spaceID, `select spaceID from spaces where space = ?`, space)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("No such space, %v", space)
		}
		return -1, err
	}
	return spaceID, nil
}

func (db *database) getInterestList(space string) (result []Interest, err error) {
	spaceID, err := db.getSpaceID(space)
	if err != nil {
		return
	}
	rows, err := db.db.Queryx(`
		select docID, state from interest
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

func (db *database) clearInterestList(space string) error {
	spaceID, err := db.getSpaceID(space)
	if err != nil {
		return err
	}
	_, err = db.db.Exec(`delete from interest where spaceID = ?`, spaceID)
	return err
}

func (db *database) resetRequested(space string) error {
	spaceID, err := db.getSpaceID(space)
	if err != nil {
		return err
	}
	_, err = db.db.Exec(`update interest set state = ? where state = ? and spaceID = ?`,
		pending, requested, spaceID)
	return err
}

func (db *database) setInterestList(space string, list []protocol.DocumentID) error {
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
	err = tx.Get(&interestCount, `select count(*) from interest where spaceID = ? and state <> ?`, spaceID, served)
	if err != nil {
		return err
	}
	if interestCount != 0 {
		return fmt.Errorf("Cannot overwrite active interest list")
	}
	_, err = tx.Exec(`delete from interest where spaceID = ?`, spaceID)
	if err != nil {
		return err
	}
	st, err := tx.Preparex(`insert into interest (spaceID, docID, state) values(?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, docID := range list {
		_, err := st.Exec(spaceID, docID, pending)
		if err != nil {
			return err
		}
	}

	now := time.Now().UnixNano()

	_, err = tx.NamedExec(`
		update spaces set listCreatedAtNanos = :now
		where spaceID = :spaceID`,

		map[string]interface{}{
			"spaceID": spaceID,
			"now":     now,
		})
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (db *database) setInterestState(space string, docID protocol.DocumentID, state InterestState) error {
	spaceID, err := db.getSpaceID(space)
	if err != nil {
		return err
	}

	_, err = db.db.Exec("update interest set state = ? where spaceID=? and docID=?", state, spaceID, docID)
	return err
}

func (db *database) GetRawDB() *sql.DB {
	return db.db.DB
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

		createSpace := `insert into spaces (space, lastUpdatedAtNanos) values(?, 0) on conflict do nothing`
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
