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
	"database/sql"
	"fmt"
	"strings"

	"github.com/erkkah/letarette/internal/snowball"
)

// ErrStemmerSettingsMismatch is returned when config and index state does not match
var ErrStemmerSettingsMismatch = fmt.Errorf("config does not match index state")

// CheckStemmerSettings verifies that the index stemmer settings match the
// current config. If there are no index settings, they will be set from the
// provided config.
func CheckStemmerSettings(db Database, cfg Config) error {
	internal := db.(*database)
	state, _, err := internal.getStemmerState()
	if err == sql.ErrNoRows {
		state = snowball.Settings{
			Stemmers:         cfg.Stemmer.Languages,
			RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
			TokenCharacters:  cfg.Stemmer.TokenCharacters,
			Separators:       cfg.Stemmer.Separators,
		}
		return internal.setStemmerState(state)
	}
	if err != nil {
		return err
	}

	stateLanguages := strings.Join(state.Stemmers, ",")
	configLanguages := strings.Join(cfg.Stemmer.Languages, ",")

	if stateLanguages != configLanguages ||
		state.RemoveDiacritics != cfg.Stemmer.RemoveDiacritics ||
		state.Separators != cfg.Stemmer.Separators ||
		state.TokenCharacters != cfg.Stemmer.TokenCharacters {
		return ErrStemmerSettingsMismatch
	}

	return nil
}
