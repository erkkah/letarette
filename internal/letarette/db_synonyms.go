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
	"fmt"
)

type Synonym struct {
	Description string
	Words       []string
}

func SetSynonyms(ctx context.Context, dbo Database, synonyms []Synonym) error {
	db := dbo.(*database)

	tx, err := db.wdb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `delete from synonym_words; delete from synonyms;`)
	if err != nil {
		return fmt.Errorf("failed to delete old synonym list: %w", err)
	}

	for _, s := range synonyms {
		res, err := tx.ExecContext(ctx, `insert into synonyms (description) values(?)`, s.Description)
		if err != nil {
			return fmt.Errorf("failed to insert new synonym: %w", err)
		}

		if rows, err := res.RowsAffected(); err != nil || rows != 1 {
			return fmt.Errorf("unexpected insert result: %v, %w", rows, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		for _, w := range s.Words {
			res, err = tx.ExecContext(ctx, `insert into synonym_words (synonymId, word) values(?, ?)`, id, w)
			if err != nil {
				return fmt.Errorf("failed to insert new word: %w", err)
			}
			if rows, err := res.RowsAffected(); err != nil || rows != 1 {
				return fmt.Errorf("unexpected insert result: %v, %w", rows, err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	tx = nil
	return nil
}
