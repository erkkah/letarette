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
	"fmt"
	"strings"
)

/*
	Dumb implementation of stop words, filtering unstemmed query phrases by stemmed stop words.
	This should better be done in the query time stemmer.
*/
func (db *database) stopwordFilterPhrases(ctx context.Context, unfiltered []Phrase) ([]Phrase, error) {
	phraseSet := map[string]Phrase{}
	phraseList := []string{}
	for _, phrase := range unfiltered {
		phraseList = append(phraseList, fmt.Sprintf("%q", phrase.Text))
		phraseSet[phrase.Text] = phrase
	}

	jsonPhrases := "[" + strings.Join(phraseList, ",") + "]"
	filterQuery := `select value from json_each(?) except select word from stopwords`
	rows, err := db.rdb.QueryxContext(ctx, filterQuery, jsonPhrases)
	if err != nil {
		return []Phrase{}, err
	}

	filtered := []Phrase{}
	for rows.Next() {
		var keptPhrase string
		err = rows.Scan(&keptPhrase)
		if err != nil {
			return filtered, err
		}
		if foundPhrase, ok := phraseSet[keptPhrase]; ok {
			filtered = append(filtered, foundPhrase)
		}
	}

	return filtered, nil
}

func (db *database) updateStopwords(ctx context.Context) error {
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
			_ = tx.Rollback()
		}
	}()

	q, err := SQL("stopwords.sql")
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		q,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	tx = nil
	return nil
}
