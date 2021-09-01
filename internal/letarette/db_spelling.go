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
	"errors"
	"fmt"
	"strings"
)

func (db *database) spellFixTerm(ctx context.Context, term string) (string, float32, bool, error) {
	var exists bool
	quotedTerm := fmt.Sprintf("%q", term)
	err := db.rdb.GetContext(ctx, &exists, `select exists(select rowid from fts where fts match ? limit 1)`, quotedTerm)
	if err != nil {
		return "", 0, false, err
	}
	if exists {
		return term, 0, false, nil
	}

	fixed := struct {
		Word     string
		Distance float32
		Score    int
	}{}
	unquotedTerm := strings.TrimSuffix(strings.TrimPrefix(term, `"`), `"`)
	err = db.rdb.GetContext(ctx, &fixed,
		`select word, distance, score from speling where word match ? limit 1`, unquotedTerm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return term, 0, false, nil
		}
		return "", 0, false, err
	}

	return fixed.Word, fixed.Distance, true, nil
}

// fixPhraseSpelling tries to spell-fix a list of phrases.
// Returns a list of possibly fixed phrases, the sum of edit distances of the fixes
// and a "fixed" status.
func (db *database) fixPhraseSpelling(ctx context.Context, phrases []Phrase) ([]Phrase, float32, bool, error) {
	clone := append(phrases[:0:0], phrases...)
	distances := float32(0.0)
	fixed := false

	nonStopwords, err := db.stopwordFilterPhrases(ctx, phrases)
	if err != nil {
		return []Phrase{}, 0, false, err
	}

	isStopword := func(phrase string) bool {
		for _, sw := range nonStopwords {
			if sw.Text == phrase {
				return false
			}
		}
		return true
	}

	for index, phrase := range phrases {
		if strings.Contains(phrase.Text, " ") {
			// Skip multi-term phrases
			continue
		}
		if isStopword(phrase.Text) {
			// Skip stopwords
			continue
		}
		if fixedPhrase, fixedDistance, phraseFixed, err := db.spellFixTerm(ctx, phrase.Text); err == nil {
			if phraseFixed {
				clone[index].Text = fixedPhrase
				fixed = true
			}
			distances += fixedDistance
		} else {
			return []Phrase{}, 0, false, err
		}
	}
	return clone, distances, fixed, nil
}
