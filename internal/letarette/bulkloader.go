// Copyright 2020 Erik Agsjö
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

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/jmoiron/sqlx"
)

func StartBulkLoad(dbo Database, space string) (*BulkLoader, error) {
	db := dbo.(*database)
	sql := db.getRawDB()
	ctx := context.Background()
	tx, err := sql.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	statement := tx.StmtxContext(ctx, db.addDocumentStatement)
	return &BulkLoader{spaceID, tx, statement, 0}, nil
}

type BulkLoader struct {
	spaceID     int
	tx          *sqlx.Tx
	statement   *sqlx.Stmt
	loadedBytes uint32
}

func (bl *BulkLoader) Load(doc protocol.Document) error {
	txt := ""
	title := ""
	if doc.Alive {
		txt = doc.Text
		title = doc.Title
	}

	bl.loadedBytes += uint32(len(title) + len(txt))

	res, err := bl.statement.Exec(
		sql.Named("spaceID", bl.spaceID),
		sql.Named("docID", doc.ID),
		sql.Named("updated", doc.Updated.UnixNano()),
		sql.Named("title", title),
		sql.Named("txt", txt),
		sql.Named("alive", doc.Alive),
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows changed")
	}
	return nil
}

func (bl *BulkLoader) Commit() error {
	return bl.tx.Commit()
}

func (bl *BulkLoader) Rollback() error {
	return bl.tx.Rollback()
}

func (bl *BulkLoader) LoadedBytes() uint32 {
	return bl.loadedBytes
}
