// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package letarette

//go:generate go run github.com/go-bindata/go-bindata/go-bindata -pkg $GOPACKAGE -o bindata.go migrations/ sql/

import (
	"crypto/rand"
	"database/sql"
	drv "database/sql/driver"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/golang-migrate/migrate/v4"
	sqlite3_migrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"

	"github.com/erkkah/letarette/internal/auxiliary"
	"github.com/erkkah/letarette/internal/compress"
	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/internal/spellfix"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

// InterestState represents the state of an interest
type InterestState int

const (
	// Waiting to be requested, unhandled
	pending InterestState = iota
	// Requested from document manager
	requested
	// Received from document manager
	served
)

// Interest represents one row in the interest list
type Interest struct {
	DocID   protocol.DocumentID `db:"docID"`
	State   InterestState
	Updated int64 `db:"updatedNanos"`
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
	Close() error
	RawQuery(string) ([]string, error)
}

type database struct {
	rdb            *sqlx.DB
	wdb            *sqlx.DB
	resultCap      int
	searchStrategy int

	addDocumentStatement    *sqlx.Stmt
	updateInterestStatement *sqlx.Stmt
}

// OpenDatabase connects to a new or existing database and
// migrates the database up to the latest version.
func OpenDatabase(cfg Config) (Database, error) {
	registerCustomDriver(cfg)
	rdb, wdb, err := openDatabase(cfg.DB.Path, cfg.Index.Spaces)
	if err != nil {
		return nil, err
	}

	if !cfg.DB.ToolConnection {
		err = preloadDB(cfg.DB.Path)
		if err != nil {
			return nil, err
		}
	}

	var addDocumentSQL string
	if cfg.Index.Compress {
		addDocumentSQL = addCompressedDocumentSQL
	} else {
		addDocumentSQL = addUncompressedDocumentSQL
	}

	addDocumentStatement, err := wdb.Preparex(addDocumentSQL)
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("failed to prepare doc update statement: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	updateInterestStatement, err := wdb.Preparex(updateInterestSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare interest update statement: %w", err)
	}

	newDB := &database{
		rdb:                     rdb,
		wdb:                     wdb,
		resultCap:               cfg.Search.Cap,
		searchStrategy:          cfg.Search.Strategy,
		addDocumentStatement:    addDocumentStatement,
		updateInterestStatement: updateInterestStatement,
	}
	return newDB, nil
}

// ResetMigration forces the migration version of a db.
// It is typically used to back out of a failed migration.
// Note: no migration steps are actually performed, it only
// sets the version and resets the dirty flag.
func ResetMigration(cfg Config, version int) error {
	registerCustomDriver(cfg)
	db, err := openMigrationConnection(cfg.DB.Path)
	if err != nil {
		return err
	}
	var current int
	err = db.Get(&current, "select version from schema_migrations")
	if err != nil {
		return err
	}
	if current < version {
		return fmt.Errorf("cannot reset migration forward from %v to %v", current, version)
	}
	_, err = db.Exec(`update schema_migrations set version=?, dirty="false"`, version)
	return err
}

func (db *database) Close() error {
	logger.Debug.Printf("Closing database")
	rErr := db.rdb.Close()

	_, tErr := db.wdb.Exec("pragma wal_checkpoint(TRUNCATE);")

	wErr := db.wdb.Close()

	if rErr != nil || tErr != nil || wErr != nil {
		return fmt.Errorf("failed to close db: %v, %v, %v", rErr, tErr, wErr)
	}
	return nil
}

func (db *database) RawQuery(statement string) ([]string, error) {
	res, err := db.rdb.Queryx(statement)
	if err != nil {
		return nil, err
	}
	err = res.Err()
	if err != nil {
		return nil, err
	}
	result := []string{}
	for res.Next() {
		row, err := res.SliceScan()
		if err != nil {
			return nil, err
		}
		colTypes, _ := res.ColumnTypes()
		var rowdata []string
		for i, col := range row {
			var coldata string
			switch colTypes[i].ScanType().Kind() {
			case reflect.String:
				coldata = fmt.Sprintf("%s", col)
			default:
				coldata = fmt.Sprintf("%v", col)
			}
			rowdata = append(rowdata, coldata)
		}
		result = append(result, strings.Join(rowdata, ", "))
	}
	return result, nil
}

func (db *database) getRawDB() *sqlx.DB {
	return db.wdb
}

func (db *database) getIndexID() (string, error) {
	var indexID string
	err := db.rdb.Get(&indexID, "select indexID from meta")
	return indexID, err
}

func initDB(db *sqlx.DB, spaces []string) error {
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

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	if dirty {
		return fmt.Errorf("Database has a dirty migration at level %v", version)
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
			return fmt.Errorf("failed to create space table: %w", err)
		}
	}

	var indexID string
	err = db.Get(&indexID, "select indexID from meta")
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get index id: %w", err)
	}
	if len(indexID) == 0 {
		var buf [128]byte
		_, err = rand.Read(buf[:])
		if err != nil {
			return fmt.Errorf("failed to generate random index id: %w", err)
		}
		high := binary.BigEndian.Uint64(buf[:64])
		low := binary.BigEndian.Uint64(buf[64:])
		indexID = fmt.Sprintf("%X%X", high, low)
		_, err = db.Exec("insert into meta (indexID) values(?)", indexID)
		if err != nil {
			return fmt.Errorf("failed to store index id: %w", err)
		}

		// Set default fts5 page size to 16k
		const pageSize = 16384
		_, err = db.Exec(`insert into fts(fts, rank) values("pgsz", ?)`, pageSize)
		if err != nil {
			return fmt.Errorf("failed to set default page size: %w", err)
		}
	}

	return nil
}

const driver = "sqlite3_letarette"

func registerCustomDriver(cfg Config) {
	drivers := sql.Drivers()
	if sort.Search(len(drivers), func(i int) bool { return drivers[i] == driver }) == len(drivers) {
		logger.Debug.Printf("Registering %q driver", driver)
		sql.Register(driver,
			&sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					logger.Debug.Printf("Initializing snowball stemmer")
					err := snowball.Init(conn, snowball.Settings{
						Stemmers:         cfg.Stemmer.Languages,
						RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
						TokenCharacters:  cfg.Stemmer.TokenCharacters,
						Separators:       cfg.Stemmer.Separators,
						MinTokenLength:   2,
					})
					if err != nil {
						return err
					}

					logger.Debug.Printf("Initializing aux functions")
					err = auxiliary.Init(conn)
					if err != nil {
						return err
					}

					logger.Debug.Printf("Initializing spellfix")
					err = spellfix.Init(conn)
					if err != nil {
						return err
					}

					logger.Debug.Printf("Initializing compress")
					err = compress.Init(conn)
					if err != nil {
						return err
					}

					logger.Debug.Printf("Setting up pragmas")
					pragmas := []string{
						"pragma threads=4",
						"pragma temp_store=2",
						"pragma wal_autocheckpoint=4000",
						fmt.Sprintf("pragma cache_size=-%d", cfg.DB.CacheSizeMB*1024),
						fmt.Sprintf("pragma mmap_size=%d", cfg.DB.MMapSizeMB*1024*1024),
					}

					_, err = conn.Exec(strings.Join(pragmas, ";"), []drv.Value{})
					if err != nil {
						return err
					}

					return nil
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
		return "", fmt.Errorf("failed to get absolute path to DB: %w", err)
	}
	escapedPath := strings.Replace(abspath, " ", "%20", -1)

	args := []string{
		"_journal=WAL",
		"_foreign_keys=true",
		"_timeout=500",
		"cache=private",
		"_mutex=no",
	}

	if mode == readOnly {
		args = append(args, []string{
			"mode=ro",
			"_query_only=true",
		}...)
	} else {
		args = append(args, []string{
			"_sync=1",
			"_rt=true",
		}...)
	}
	return fmt.Sprintf("file:%s?%s", escapedPath, strings.Join(args, "&")), nil
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
	if err != nil {
		return nil, nil, err
	}
	rdb, err = sqlx.Connect(driver, readSqliteURL)
	if err != nil {
		return
	}
	rdb.SetMaxOpenConns(0)
	rdb.SetMaxIdleConns(8)

	err = initDB(wdb, spaces)
	if err != nil {
		return
	}

	return
}

// preloadDB reads one byte from each page of the database
// file to force it into the system cache.
func preloadDB(dbPath string) error {
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		return err
	}
	file, err := os.Open(dbPath)
	if err != nil {
		return err
	}

	fileSize := fileInfo.Size()
	pageSize := int64(os.Getpagesize())
	buf := make([]byte, 1)

	fd := file.Fd()
	err = fadvice(fd, fileSize)
	if err != nil {
		logger.Warning.Printf("Failed to advice about file usage: %v", err)
	}

	go func() {
		defer file.Close()
		logger.Info.Printf("Pre-loading database start")
		for pos := int64(0); pos < fileSize; pos += pageSize {
			_, err = file.ReadAt(buf, pos)
			if err != nil {
				logger.Error.Printf("Pre-loading db failed: %v", err)
				break
			}
		}
		logger.Info.Printf("Pre-loading database done")
	}()

	return nil
}
