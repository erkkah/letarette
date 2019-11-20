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
	"strings"
)

func (db *database) spellFixTerm(ctx context.Context, term string) (string, bool, error) {
	var exists bool
	err := db.rdb.GetContext(ctx, &exists, `select exists(select rowid from fts where fts match ? limit 1)`, term)
	if err != nil {
		return "", false, err
	}
	if exists {
		return term, false, nil
	}

	fixed := struct {
		Word  string
		Score int
	}{}
	unquotedTerm := strings.TrimSuffix(strings.TrimPrefix(term, `"`), `"`)
	err = db.rdb.GetContext(ctx, &fixed, `select word, score from speling where word match ? limit 1`, unquotedTerm)
	if err != nil {
		if err == sql.ErrNoRows {
			return term, false, nil
		}
		return "", false, err
	}
	// ??? Use score?
	return fixed.Word, true, nil
}

func (db *database) fixPhraseSpelling(ctx context.Context, phrases []Phrase) ([]Phrase, bool, error) {
	clone := append(phrases[:0:0], phrases...)
	fixed := false
	for index, phrase := range phrases {
		if strings.Contains(phrase.Text, " ") {
			// Skip multi-term phrases
			continue
		}
		if fixedPhrase, phraseFixed, err := db.spellFixTerm(ctx, phrase.Text); err == nil {
			if phraseFixed {
				clone[index].Text = fixedPhrase
				fixed = true
			}
		} else {
			return []Phrase{}, false, err
		}
	}
	return clone, fixed, nil
}
