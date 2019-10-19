package letarette

//go:generate go-bindata -pkg $GOPACKAGE -o migrations.go migrations/

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/golang-migrate/migrate/v4"
	sqlite3_migrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"

	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/logger"
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

	addDocumentUpdate(ctx context.Context, doc protocol.Document) error
	commitInterestList(ctx context.Context, space string) error
	getLastUpdateTime(context.Context, string) (time.Time, error)

	clearInterestList(context.Context, string) error
	resetRequested(context.Context, string) error

	setInterestList(context.Context, string, []protocol.DocumentID) error
	getInterestList(context.Context, string) ([]Interest, error)
	setInterestState(context.Context, string, protocol.DocumentID, InterestState) error

	getInterestListState(context.Context, string) (InterestListState, error)

	search(ctx context.Context, phrase string, spaces []string, limit uint16, offset uint16) ([]protocol.SearchResult, error)

	getStemmerState() (snowball.Settings, time.Time, error)
	setStemmerState(snowball.Settings) error

	getRawDB() *sqlx.DB
}

type database struct {
	rdb *sqlx.DB
	wdb *sqlx.DB
}

// OpenDatabase connects to a new or existing database and
// migrates the database up to the latest version.
func OpenDatabase(cfg Config) (Database, error) {
	registerSnowballDriver(cfg)
	rdb, wdb, err := openDatabase(cfg.Db.Path, cfg.Index.Spaces)
	if err != nil {
		return nil, err
	}

	newDB := &database{rdb, wdb}
	return newDB, nil
}

func ResetMigration(cfg Config, version int) error {
	registerSnowballDriver(cfg)
	db, err := openMigrationConnection(cfg.Db.Path)
	if err != nil {
		return err
	}
	var current int
	err = db.Get(&current, "select version from schema_migrations")
	if err != nil {
		return err
	}
	if current < version {
		return fmt.Errorf("Cannot reset migration forward from %v to %v", current, version)
	}
	_, err = db.Exec(`update schema_migrations set version=?, dirty="false"`, version)
	return err
}

func (db *database) Close() {
	db.rdb.Close()
	db.wdb.Close()
}

func (db *database) getRawDB() *sqlx.DB {
	return db.wdb
}

func (db *database) getLastUpdateTime(ctx context.Context, space string) (t time.Time, err error) {
	var timestampNanos int64
	err = db.rdb.GetContext(ctx, &timestampNanos, "select lastUpdatedAtNanos from spaces where space = ?", space)
	t = time.Unix(0, timestampNanos)
	return
}

func (db *database) addDocumentUpdate(ctx context.Context, doc protocol.Document) error {
	spaceID, err := db.getSpaceID(ctx, doc.Space)
	if err != nil {
		return err
	}

	tx, err := db.wdb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	txt := ""
	if doc.Alive {
		txt = doc.Text
	}
	res, err := tx.ExecContext(ctx, `replace into docs (spaceID, docID, updatedNanos, txt, alive) values (?, ?, ?, ?, ?)`,
		spaceID, doc.ID, doc.Updated.UnixNano(), txt, doc.Alive)

	if err != nil {
		return fmt.Errorf("Failed to update doc: %w", err)
	}

	updatedRows, _ := res.RowsAffected()
	if updatedRows != 1 {
		return fmt.Errorf("Failed to update index, no rows affected")
	}

	_, err = tx.ExecContext(ctx, `update interest set state=? where spaceID=? and docID=?`, served, spaceID, doc.ID)

	if err != nil {
		return fmt.Errorf("Failed to update interest list: %w", err)
	}

	err = tx.Commit()
	if err == nil {
		tx = nil
	}

	return err
}

func (db *database) commitInterestList(ctx context.Context, space string) error {
	tx, err := db.wdb.BeginTxx(ctx, nil)
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

	err = tx.GetContext(ctx, &indexPosition, `
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

	res, err := tx.ExecContext(ctx, "update spaces set lastUpdatedAtNanos = ?, lastUpdatedDocID = ? where space = ?",
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

func (db *database) getInterestListState(ctx context.Context, space string) (state InterestListState, err error) {
	err = db.rdb.GetContext(ctx, &state,
		`select listCreatedAtNanos, lastUpdatedAtNanos, lastUpdatedDocID from spaces where space = ?`, space)
	return
}

func (db *database) getSpaceID(ctx context.Context, space string) (int, error) {
	var spaceID int
	err := db.rdb.GetContext(ctx, &spaceID, `select spaceID from spaces where space = ?`, space)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("No such space, %v", space)
		}
		return -1, err
	}
	return spaceID, nil
}

func (db *database) getInterestList(ctx context.Context, space string) (result []Interest, err error) {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return
	}
	rows, err := db.rdb.QueryxContext(ctx,
		`
		select docID, state from interest
		where spaceID = ?
		`, spaceID)
	if err != nil {
		return
	}
	defer rows.Close()

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

func (db *database) clearInterestList(ctx context.Context, space string) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}
	_, err = db.wdb.ExecContext(ctx, `delete from interest where spaceID = ?`, spaceID)
	return err
}

func (db *database) resetRequested(ctx context.Context, space string) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}
	_, err = db.wdb.ExecContext(ctx, `update interest set state = ? where state = ? and spaceID = ?`,
		pending, requested, spaceID)
	return err
}

func (db *database) setInterestList(ctx context.Context, space string, list []protocol.DocumentID) error {

	tx, err := db.wdb.BeginTxx(ctx, nil)
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var spaceID int
	err = tx.GetContext(ctx, &spaceID, `select spaceID from spaces where space = ?`, space)
	if err != nil {
		return err
	}
	var interestCount int
	err = tx.GetContext(ctx, &interestCount, `select count(*) from interest where spaceID = ? and state <> ?`, spaceID, served)
	if err != nil {
		return err
	}
	if interestCount != 0 {
		return fmt.Errorf("Cannot overwrite active interest list")
	}
	_, err = tx.ExecContext(ctx, `delete from interest where spaceID = ?`, spaceID)
	if err != nil {
		return err
	}
	st, err := tx.PreparexContext(ctx, `insert into interest (spaceID, docID, state) values(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer st.Close()

	for _, docID := range list {
		_, err := st.ExecContext(ctx, spaceID, docID, pending)
		if err != nil {
			return err
		}
	}

	now := time.Now().UnixNano()

	_, err = tx.NamedExecContext(ctx,
		`
		update spaces set listCreatedAtNanos = :now
		where spaceID = :spaceID
		`,

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

func (db *database) setInterestState(ctx context.Context, space string, docID protocol.DocumentID, state InterestState) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}

	_, err = db.wdb.ExecContext(ctx, "update interest set state = ? where spaceID=? and docID=?", state, spaceID, docID)
	return err
}

func (db *database) search(ctx context.Context, phrase string, spaces []string, limit uint16, offset uint16) ([]protocol.SearchResult, error) {
	const left = "\u3016"
	const right = "\u3017"
	const ellipsis = "\u2026"

	query := fmt.Sprintf(`
	select docs.docID as id, replace(snippet(fts, 0, ?, ?, ?, 8), X'0A', " ") as snippet, rank
	from fts join docs on fts.rowid = docs.id
	where fts match '"%s"' order by rank asc limit ? offset ?
	`, phrase)
	logger.Debug.Printf("Search query: [%s]", query)

	var result []protocol.SearchResult
	err := db.rdb.SelectContext(ctx, &result, query,
		left, right, ellipsis, limit, offset)

	return result, err
}

func (db *database) getStemmerState() (snowball.Settings, time.Time, error) {
	query := `
	select
	languages,
	removeDiacritics as removediacritics,
	tokenCharacters as tokencharacters,
	separators,
	updated
	from stemmerstate
	`
	var state struct {
		Languages string
		Updated   time.Time
		snowball.Settings
	}
	err := db.rdb.Get(&state, query)
	if len(state.Languages) == 0 {
		state.Stemmers = []string{}
	} else {
		state.Stemmers = strings.Split(state.Languages, ",")
	}
	return state.Settings, state.Updated, err
}

func (db *database) setStemmerState(state snowball.Settings) error {
	_, _, err := db.getStemmerState()

	if err == sql.ErrNoRows {
		insert := `
		insert into stemmerstate (languages, removeDiacritics, tokenCharacters, separators)
		values ("", "false", "", "");
		`
		result, err := db.wdb.Exec(insert)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return fmt.Errorf("Failed to insert default state")
		}
	}
	query := `
	update stemmerstate
	set languages = ?, removeDiacritics = ?, tokenCharacters = ?, separators = ?
	`

	languages := strings.Join(state.Stemmers, ",")
	_, err = db.wdb.Exec(query,
		languages,
		state.RemoveDiacritics,
		state.TokenCharacters,
		state.Separators,
	)
	return err
}

func initDB(db *sqlx.DB, sqliteURL string, spaces []string) error {
	migrations, err := AssetDir("migrations")
	if err != nil {
		return err
	}
	res := bindata.Resource(migrations, func(name string) ([]byte, error) {
		return Asset("migrations/" + name)
	})

	sourceDriver, err := bindata.WithInstance(res)
	if err != nil {
		return err
	}

	dbDriver, err := sqlite3_migrate.WithInstance(db.DB, &sqlite3_migrate.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("go-bindata", sourceDriver, "letarette", dbDriver)
	if err != nil {
		return err
	}

	version, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	runMigration := version == 0

	if !runMigration {
		next, err := sourceDriver.Next(version)
		if err == nil {
			runMigration = next > version
		} else {
			// The source driver should return ErrNotExist
			_, isPathError := err.(*os.PathError)

			if !isPathError && err != os.ErrNotExist {
				return err
			}
		}
	}

	if runMigration {
		logger.Info.Printf("Applying migrations")
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			return err
		}
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

const driver = "sqlite3_snowball"

func registerSnowballDriver(cfg Config) {
	drivers := sql.Drivers()
	if sort.Search(len(drivers), func(i int) bool { return drivers[i] == driver }) == len(drivers) {
		logger.Debug.Printf("Registering %q driver", driver)
		sql.Register(driver,
			&sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					logger.Debug.Printf("Initializing snowball stemmer")
					return snowball.Init(conn, snowball.Settings{
						Stemmers:         cfg.Stemmer.Languages,
						RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
						TokenCharacters:  cfg.Stemmer.TokenCharacters,
						Separators:       cfg.Stemmer.Separators,
					})
				},
			})
	}
}

type connectionMode bool

const (
	readOnly  connectionMode = true
	readWrite connectionMode = false
)

func getDatabaseURL(dbPath string, mode connectionMode) (string, error) {
	abspath, err := filepath.Abs(dbPath)
	if err != nil {
		return "", fmt.Errorf("Failed to get absolute path to DB: %w", err)
	}
	escapedPath := strings.Replace(abspath, " ", "%20", -1)

	if mode == readOnly {
		return fmt.Sprintf("file:%s?_journal=WAL&_query_only=true&_foreign_keys=true&_timeout=500&cache=shared", escapedPath), nil
	}
	return fmt.Sprintf("file:%s?_journal=WAL&_foreign_keys=true&_timeout=500&cache=private", escapedPath), nil
}

func openMigrationConnection(dbPath string) (db *sqlx.DB, err error) {
	url, err := getDatabaseURL(dbPath, readWrite)
	if err != nil {
		return nil, err
	}
	db, err = sqlx.Connect(driver, url)
	return
}

func openDatabase(dbPath string, spaces []string) (rdb *sqlx.DB, wdb *sqlx.DB, err error) {

	// Only one writer
	writeSqliteURL, err := getDatabaseURL(dbPath, readWrite)
	if err != nil {
		return nil, nil, err
	}
	wdb, err = sqlx.Connect(driver, writeSqliteURL)
	if err != nil {
		return
	}
	wdb.SetMaxOpenConns(1)

	// Multiple readers
	readSqliteURL, err := getDatabaseURL(dbPath, readOnly)
	rdb, err = sqlx.Connect(driver, readSqliteURL)
	if err != nil {
		return
	}

	if len(spaces) < 1 {
		return nil, nil, fmt.Errorf("No spaces defined: %v", spaces)
	}
	err = initDB(wdb, writeSqliteURL, spaces)
	return
}
