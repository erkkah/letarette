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
	"strings"
	"time"

	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/protocol"
)

func (db *database) getLastUpdateTime(ctx context.Context, space string) (t time.Time, err error) {
	var timestampNanos int64
	err = db.rdb.GetContext(ctx, &timestampNanos, "select lastUpdatedAtNanos from spaces where space = ?", space)
	t = time.Unix(0, timestampNanos)
	return
}

func (db *database) getDocumentCount(ctx context.Context) (uint64, error) {
	var count uint64
	err := db.rdb.GetContext(ctx, &count, "select count(*) from docs")
	return count, err
}

var addCompressedDocumentSQL = `
replace into docs (spaceID, docID, updatedNanos, title, txt, alive)
values (:spaceID, :docID, :updated, :title, compress(:txt), :alive);
`

var addUncompressedDocumentSQL = `
replace into docs (spaceID, docID, updatedNanos, title, txt, alive)
values (:spaceID, :docID, :updated, :title, :txt, :alive);
`

var updateInterestSQL = `
update interest set state=:state where spaceID=:spaceID and docID=:docID
`

func (db *database) addDocumentUpdates(ctx context.Context, space string, docs []protocol.Document) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}

	tx, err := db.wdb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	docsStatement := tx.StmtxContext(ctx, db.addDocumentStatement)
	interestStatement := tx.StmtxContext(ctx, db.updateInterestStatement)

	for _, doc := range docs {
		txt := ""
		title := ""
		if doc.Alive {
			txt = doc.Text
			title = doc.Title
		}

		res, err := docsStatement.ExecContext(
			ctx,
			sql.Named("spaceID", spaceID),
			sql.Named("docID", doc.ID),
			sql.Named("updated", doc.Updated.UnixNano()),
			sql.Named("title", title),
			sql.Named("txt", txt),
			sql.Named("alive", doc.Alive),
		)

		if err != nil {
			return fmt.Errorf("failed to update doc: %w", err)
		}

		updatedRows, _ := res.RowsAffected()
		if updatedRows != 1 {
			return fmt.Errorf("failed to update index, no rows affected")
		}

		_, err = interestStatement.ExecContext(
			ctx,
			sql.Named("state", served),
			sql.Named("spaceID", spaceID),
			sql.Named("docID", doc.ID),
		)

		if err != nil {
			return fmt.Errorf("failed to update interest list: %w", err)
		}
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
			_ = tx.Rollback()
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
		select docs.updatedNanos, docs.docID
		from
		interest
		left join docs using(docID)
		cross join listState
		where interest.state = ? and docs.updatedNanos <= listState.listCreatedAtNanos
		order by docs.updatedNanos desc, docs.docID desc
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
		err = fmt.Errorf("failed to update index position for space %q", space)
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
			err = fmt.Errorf("no such space, %v", space)
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
	err = rows.Err()
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

func (db *database) fakeServeRequested(ctx context.Context, space string) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}
	_, err = db.wdb.ExecContext(ctx, `update interest set state = ? where state = ? and spaceID = ?`,
		served, requested, spaceID)
	return err
}

func (db *database) hasDocument(ctx context.Context, space string, doc Interest) (bool, error) {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return false, err
	}
	var exists bool
	err = db.rdb.GetContext(
		ctx, &exists,
		`select count(*) == 1 from docs where docs.spaceID = ? and docs.docID = ? and docs.updatedNanos = ?`,
		spaceID,
		doc.DocID,
		doc.Updated,
	)
	return exists, err
}

func (db *database) setInterestList(ctx context.Context, indexUpdate protocol.IndexUpdate) error {

	tx, err := db.wdb.BeginTxx(ctx, nil)
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var spaceID int
	err = tx.GetContext(ctx, &spaceID, `select spaceID from spaces where space = ?`, indexUpdate.Space)
	if err != nil {
		return err
	}
	var interestCount int
	err = tx.GetContext(ctx, &interestCount,
		`select count(*) from interest where spaceID = ? and state <> ?`, spaceID, served)
	if err != nil {
		return err
	}
	if interestCount != 0 {
		err = fmt.Errorf("cannot overwrite active interest list")
		return err
	}
	_, err = tx.ExecContext(ctx, `delete from interest where spaceID = ?`, spaceID)
	if err != nil {
		return err
	}
	st, err := tx.PreparexContext(ctx, `insert into interest (spaceID, docID, state, updatedNanos) values(?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer st.Close()

	for _, update := range indexUpdate.Updates {
		_, err := st.ExecContext(ctx, spaceID, update.ID, pending, update.Updated.UnixNano())
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

func (db *database) setInterestState(
	ctx context.Context, space string, docID protocol.DocumentID, state InterestState,
) error {
	spaceID, err := db.getSpaceID(ctx, space)
	if err != nil {
		return err
	}

	_, err = db.wdb.ExecContext(ctx, "update interest set state = ? where spaceID=? and docID=?", state, spaceID, docID)
	return err
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
			return fmt.Errorf("failed to insert default state")
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
