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

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/erkkah/letarette/internal/snowball"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo linux LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo windows LDFLAGS: -L${SRCDIR}/.. -lsqlite
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #include <sqlite3-binding.h>
import "C"

// Stats holds statistics gathered by GetIndexStats
type Stats struct {
	Spaces []struct {
		Name  string
		State InterestListState
	}
	CommonTerms []struct {
		Term  string
		Count int
	}
	TotalTerms  int
	UniqueTerms int
	Docs        int
	Stemmer     snowball.Settings
}

// GetIndexStats collects statistics about the index,
// partly by the use of the fts5vocab virtual table.
func GetIndexStats(dbo Database) (Stats, error) {
	var s Stats
	db := dbo.(*database)

	sql := db.getRawDB()

	var err error

	ctx := context.Background()
	conn, err := sql.Conn(ctx)
	if err != nil {
		return s, err
	}
	defer conn.Close()

	s.Stemmer, _, _ = db.getStemmerState()

	rows, err := conn.QueryContext(ctx, `select space from spaces`)
	if err != nil {
		return s, err
	}
	err = rows.Err()
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var space string
		_ = rows.Scan(&space)

		state, err := db.getInterestListState(ctx, space)
		if err != nil {
			return s, err
		}
		s.Spaces = append(s.Spaces, struct {
			Name  string
			State InterestListState
		}{space, state})
	}

	_, err = conn.ExecContext(
		ctx,
		`create virtual table temp.rowstats using fts5vocab(main, 'fts', 'row');`,
	)
	if err != nil {
		return s, err
	}

	_, err = conn.ExecContext(
		ctx,
		`create virtual table temp.instancestats using fts5vocab(main, 'fts', 'instance');`,
	)
	if err != nil {
		return s, err
	}

	rows, err = conn.QueryContext(
		ctx,
		`select term, cnt from temp.rowstats order by cnt desc limit 15;`,
	)
	if err != nil {
		return s, err
	}
	err = rows.Err()
	if err != nil {
		return s, err
	}

	for rows.Next() {
		var term string
		var count int
		err = rows.Scan(&term, &count)
		if err != nil {
			return s, err
		}
		s.CommonTerms = append(s.CommonTerms, struct {
			Term  string
			Count int
		}{term, count})
	}

	row := conn.QueryRowContext(
		ctx,
		`select count(term) from temp.rowstats`,
	)
	_ = row.Scan(&s.UniqueTerms)

	row = conn.QueryRowContext(
		ctx,
		`select count(term) from temp.instancestats`,
	)
	_ = row.Scan(&s.TotalTerms)

	row = conn.QueryRowContext(
		ctx,
		`select count(*) from docs`,
	)
	_ = row.Scan(&s.Docs)

	return s, nil
}

// CheckIndex runs an integrity check on the index
func CheckIndex(dbo Database) error {
	db := dbo.(*database)
	sql := db.getRawDB()
	_, err := sql.Exec(`insert into fts(fts) values("integrity-check");`)
	if err != nil {
		return err
	}
	return nil
}

// RebuildIndex rebuilds the fts index from the docs table
func RebuildIndex(dbo Database) error {
	db := dbo.(*database)
	sql := db.getRawDB()
	_, err := sql.Exec(`insert into fts(fts) values("rebuild");`)
	if err != nil {
		return err
	}
	return nil
}

// VacuumIndex runs vacuum on the database to reclaim space
func VacuumIndex(dbo Database) error {
	db := dbo.(*database)
	sql := db.getRawDB()
	_, err := sql.Exec(`vacuum`)
	if err != nil {
		return err
	}
	return nil
}

// IndexOptimizer is used to run step-wise index optimization.
// The instance must be closed by calling Close() to return
// the database connection to the pool.
type IndexOptimizer struct {
	conn          *sql.Conn
	ctx           context.Context
	pageIncrement int
}

// Step runs one step of the optimizer.
// Returns true when optimization is complete.
// Stopping before done is OK.
func (o IndexOptimizer) Step() (bool, error) {
	changesBefore, err := o.totalChanges()
	if err != nil {
		return false, err
	}
	_, err = o.conn.ExecContext(o.ctx, `insert into fts(fts, rank) values("merge", ?);`, o.pageIncrement)
	if err != nil {
		return false, err
	}
	changesAfter, err := o.totalChanges()
	if err != nil {
		return false, err
	}
	return changesAfter-changesBefore < 2, nil
}

// Close returns the database connection to the pool.
func (o IndexOptimizer) Close() error {
	return o.conn.Close()
}

func dbFromConnection(conn *sqlite3.SQLiteConn) *C.sqlite3 {
	dbVal := reflect.ValueOf(conn).Elem().FieldByName("db")
	dbPtr := unsafe.Pointer(dbVal.Pointer())
	return (*C.sqlite3)(dbPtr)
}

func (o IndexOptimizer) totalChanges() (int, error) {
	var changes int

	err := o.conn.Raw(func(conn interface{}) error {
		if driverConn, ok := conn.(*sqlite3.SQLiteConn); ok {
			var sqliteConn *C.sqlite3 = dbFromConnection(driverConn)
			changes = int(C.sqlite3_total_changes(sqliteConn))
			return nil
		}
		return fmt.Errorf("unsupported driver")
	})

	return changes, err
}

// StartIndexOptimization initiates a step-wise index optimization and returns
// an IndexOptimizer instance on success.
func StartIndexOptimization(dbo Database, pageIncrement int) (*IndexOptimizer, error) {
	db := dbo.(*database)
	sql := db.getRawDB()
	ctx := context.Background()
	conn, err := sql.Conn(ctx)
	if err != nil {
		return nil, err
	}

	_, err = conn.ExecContext(ctx, `insert into fts(fts, rank) values("merge", ?);`, -pageIncrement)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &IndexOptimizer{
		conn:          conn,
		ctx:           ctx,
		pageIncrement: pageIncrement,
	}, nil
}

// ForceIndexStemmerState resets the stemmer state stored in the database
// to the provided state.
func ForceIndexStemmerState(state snowball.Settings, dbo Database) error {
	db := dbo.(*database)
	return db.setStemmerState(state)
}

// SetIndexPageSize sets the max page size for future index allocations.
func SetIndexPageSize(dbo Database, pageSize int) error {
	db := dbo.(*database)
	sql := db.getRawDB()
	_, err := sql.Exec(`insert into fts(fts, rank) values("pgsz", ?)`, pageSize)
	return err
}

// CompressIndex compresses the txt column
func CompressIndex(ctx context.Context, dbo Database) error {
	db := dbo.(*database)
	sql := db.getRawDB()

	conn, err := sql.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `update docs set txt=compress(txt) where not iscompressed(txt)`)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	tx = nil
	return nil
}

// GetSpellfixLag returns how many words in the main index that are not yet in the spelling index.
func GetSpellfixLag(ctx context.Context, dbo Database, minCount int) (int, error) {
	db := dbo.(*database)
	rawdb := db.getRawDB()
	conn, err := rawdb.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	_, err = conn.ExecContext(
		ctx,
		`create virtual table if not exists temp.stats using fts5vocab(main, 'fts', 'row');`,
	)
	if err != nil {
		return 0, err
	}

	row := conn.QueryRowContext(
		ctx,
		`
		with
		allwords as(
			select count(term) as wordcount
			from temp.stats
			where length(term) > 3
			and cnt >= ?
		),
		spellwords as (
			select count(*) as cnt from speling
		)
		select (select wordcount from allwords) - (select cnt from spellwords) as lag
		`,
		minCount,
	)

	var lag int
	err = row.Scan(&lag)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return lag, nil
}

// UpdateSpellfix updates the spelling table with the top terms
// from the fts.
func UpdateSpellfix(ctx context.Context, dbo Database, minCount int) error {
	db := dbo.(*database)
	sql := db.getRawDB()
	conn, err := sql.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(
		ctx,
		`create virtual table if not exists temp.stats using fts5vocab(main, 'fts', 'row');`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`delete from speling`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		insert into speling(word, rank)
		select term, cnt from temp.stats
		where length(term) > 3
		and cnt >= ?
		order by cnt desc
		`,
		minCount,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	tx = nil
	return nil
}
