package letarette

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/erkkah/letarette/internal/snowball"
)

// ErrStemmerSettingsMismatch is returned when config and index state does not match
var ErrStemmerSettingsMismatch = fmt.Errorf("Config does not match index state")

// CheckStemmerSettings verifies that the index stemmer settings match the
// current config. If there are no index settings, they will be set from the
// provided config.
func CheckStemmerSettings(db Database, cfg Config) error {
	state, _, err := db.getStemmerState()
	if err == sql.ErrNoRows {
		state := snowball.Settings{
			Stemmers:         cfg.Stemmer.Languages,
			RemoveDiacritics: cfg.Stemmer.RemoveDiacritics,
			TokenCharacters:  cfg.Stemmer.TokenCharacters,
			Separators:       cfg.Stemmer.Separators,
		}
		return db.setStemmerState(state)
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
