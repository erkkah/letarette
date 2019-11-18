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
// #cgo darwin CFLAGS: -I/usr/local/opt/sqlite/include
// #cgo LDFLAGS: -lsqlite3
// #cgo darwin LDFLAGS: -L/usr/local/opt/sqlite/lib
// #include <sqlite3.h>
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
	Terms   int
	Docs    int
	Stemmer snowball.Settings
}

// GetIndexStats collects statistics about the index,
// partly by the use of the fts4vocab virtual table.
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
	for rows.Next() {
		var space string
		rows.Scan(&space)

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
		`create virtual table temp.stats using fts5vocab(main, 'fts', 'row');`,
	)
	if err != nil {
		return s, err
	}

	rows, err = conn.QueryContext(
		ctx,
		`select term, sum(cnt) as num from temp.stats group by term order by num desc limit 15;`,
	)
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
		`select count(distinct term) from temp.stats`,
	)
	row.Scan(&s.Terms)

	row = conn.QueryRowContext(
		ctx,
		`select count(*) from docs`,
	)
	row.Scan(&s.Docs)

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
		return fmt.Errorf("Unsupported driver")
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
		conn.Close()
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
