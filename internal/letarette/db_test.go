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
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/protocol"

	gta "gotest.tools/assert"
)

type testSetup struct {
	tmpDir string
	config Config
	db     *database
}

func (setup *testSetup) cleanup() {
	if setup.db != nil {
		setup.db.Close()
	}
	err := os.RemoveAll(setup.tmpDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to delete test temp dir: %v", err))
	}
}

func getTestSetup(t *testing.T, compress ...bool) *testSetup {
	setup := new(testSetup)
	var err error
	setup.tmpDir, err = ioutil.TempDir("", "letarette")
	if err != nil {
		t.Fatal("Failed to create test temp dir")
	}
	setup.config.Db.Path = path.Join(setup.tmpDir, "leta.db")
	setup.config.Index.Spaces = []string{"test"}
	for _, state := range compress {
		setup.config.Index.Compress = state
	}

	setup.config.Stemmer.Languages = []string{"english"}

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	setup.db = db.(*database)

	return setup
}

func TestOpen(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	gta.Assert(t, setup.db != nil, "Database is nil!")
}

func TestAddDocument_EmptySpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	docs := []protocol.Document{
		protocol.Document{},
	}
	err := setup.db.addDocumentUpdates(ctx, "", docs)
	gta.ErrorContains(t, err, "No such space", "Adding document with empty space should fail")
}

func TestAddDocument_NewDocument(t *testing.T) {
	setup := getTestSetup(t, false)
	defer setup.cleanup()

	// "Cortex " below (note the space) is a "valid"
	// compressed header in the original sqlite3 compress extension.
	docs := []protocol.Document{
		protocol.Document{
			ID:      "myID",
			Updated: time.Now(),
			Text:    "Cortex and such",
			Alive:   true,
		},
	}
	ctx := context.Background()
	err := setup.db.addDocumentUpdates(ctx, "test", docs)

	gta.NilError(t, err, "Failed to add new document")
}

func TestAddDocument_NewDocument_Compressed(t *testing.T) {
	setup := getTestSetup(t, true)
	defer setup.cleanup()

	docs := []protocol.Document{
		protocol.Document{
			ID:      "myID",
			Updated: time.Now(),
			Text:    "tjo och hej",
			Alive:   true,
		},
	}
	ctx := context.Background()
	err := setup.db.addDocumentUpdates(ctx, "test", docs)

	gta.NilError(t, err, "Failed to add new document")
}

func TestCommitInterestList_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	err := setup.db.commitInterestList(ctx, "test")
	gta.NilError(t, err, "Failed to commit empty list")
}

func TestCommitInterestList_NonEmptyNoUpdates(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	beforeState, err := setup.db.getInterestListState(ctx, "test")
	gta.NilError(t, err, "Failed to get list state")

	list := protocol.IndexUpdate{
		Space: "test",
		Updates: []protocol.DocumentReference{
			{
				ID: "bello", Updated: time.Now(),
			},
			{
				ID: "koko", Updated: time.Now(),
			},
		},
	}

	err = setup.db.setInterestList(ctx, list)
	gta.NilError(t, err, "Setting interest list failed")

	err = setup.db.commitInterestList(ctx, "test")
	gta.NilError(t, err, "Failed to commit list")

	afterState, err := setup.db.getInterestListState(ctx, "test")
	gta.NilError(t, err, "Failed to get list state")

	gta.Assert(t, beforeState.LastUpdated == afterState.LastUpdated, "Expected untouched state")
	gta.Assert(t, beforeState.LastUpdatedDocID == afterState.LastUpdatedDocID, "Expected untouched state")
}

func TestCommitInterestList_NonEmptyWithUpdates(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := protocol.IndexUpdate{
		Space: "test",
		Updates: []protocol.DocumentReference{
			{
				ID: "bello", Updated: time.Now(),
			},
			{
				ID: "koko", Updated: time.Now(),
			},
		},
	}

	docTime := time.Now()
	docID := protocol.DocumentID("koko")
	docs := []protocol.Document{
		protocol.Document{
			ID:      docID,
			Updated: docTime,
			Text:    "tjo och hej",
			Alive:   true,
		},
	}

	ctx := context.Background()
	err := setup.db.setInterestList(ctx, list)
	gta.NilError(t, err, "Setting interest list failed: %v", err)

	err = setup.db.addDocumentUpdates(ctx, "test", docs)
	gta.NilError(t, err, "Failed to add document: %v", err)

	err = setup.db.commitInterestList(ctx, "test")
	gta.NilError(t, err, "Failed to commit list: %v", err)

	afterState, err := setup.db.getInterestListState(ctx, "test")
	gta.NilError(t, err, "Failed to get list state: %v", err)

	if docTime.UnixNano() != afterState.LastUpdated {
		t.Errorf("Expected last updated to be %v, was %v", docTime.UnixNano(), afterState.LastUpdated)
	}
	if docID != afterState.LastUpdatedDocID {
		t.Errorf("Expected last updated ID to be %v, was %v", docID, afterState.LastUpdatedDocID)
	}
}

func TestGetLastUpdateTime_ExistingSpace(t *testing.T) {
	then := time.Unix(1, 0)
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	last, err := setup.db.getLastUpdateTime(ctx, "test")
	gta.NilError(t, err, "Failed to get last update time: %v", err)
	if !last.Before(then) {
		t.Errorf("Initial update time should be before %v, got %v", then, last)
	}
}

func TestGetLastUpdateTime_NonExistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	_, err := setup.db.getLastUpdateTime(ctx, "popowkqd")
	gta.ErrorContains(t, err, "sql: no rows", "Fetching last update time for unknown space should fail!")
}

func TestGetInterestList_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	list, err := setup.db.getInterestList(ctx, "test")
	gta.NilError(t, err, "Failed to get interest list: %v", err)
	gta.Assert(t, len(list) == 0, "Length should be empty")
}

func TestGetInterestList_NonexistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	ctx := context.Background()
	_, err := setup.db.getInterestList(ctx, "kawonka")
	gta.ErrorContains(t, err, "No such space", "Fetching interest list for nonexisting space should fail!")
}

func TestSetInterestList_NonexistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := protocol.IndexUpdate{
		Space: "kawonka",
		Updates: []protocol.DocumentReference{

			{
				ID: "koko", Updated: time.Now(),
			},
		},
	}

	ctx := context.Background()
	err := setup.db.setInterestList(ctx, list)
	gta.ErrorContains(t, err, "sql: no rows", "Setting interest list for nonexisting space should fail!")
}

func TestSetGetInterestList_CurrentListEmpty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := protocol.IndexUpdate{
		Space: "test",
		Updates: []protocol.DocumentReference{
			{
				ID: "bello", Updated: time.Now(),
			},
			{
				ID: "koko", Updated: time.Now(),
			},
		},
	}

	ctx := context.Background()
	err := setup.db.setInterestList(ctx, list)
	gta.NilError(t, err, "Setting interest list failed: %v", err)

	fetchedSlice, err := setup.db.getInterestList(ctx, "test")
	gta.NilError(t, err, "Getting interest list failed: %v", err)
	sort.Slice(fetchedSlice, func(i int, j int) bool {
		return fetchedSlice[i].DocID < fetchedSlice[j].DocID
	})

	for i, interest := range fetchedSlice {
		if interest.State == served {
			t.Errorf("New interest should be unserved")
		}
		if interest.DocID != list.Updates[i].ID {
			t.Errorf("New list does not match")
		}
	}
}

func TestSetInterestList_CurrentListNonEmpty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := protocol.IndexUpdate{
		Space: "test",
		Updates: []protocol.DocumentReference{
			{
				ID: "bello", Updated: time.Now(),
			},
			{
				ID: "koko", Updated: time.Now(),
			},
		},
	}

	ctx := context.Background()
	err := setup.db.setInterestList(ctx, list)
	gta.NilError(t, err, "Setting interest list failed: %v", err)

	err = setup.db.setInterestList(ctx, list)
	gta.ErrorContains(t, err, "Cannot overwrite", "Setting interest list with current list should fail!")
}

func TestGetStemmerState_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	_, _, err := setup.db.getStemmerState()
	gta.Assert(t, err == sql.ErrNoRows, "Expected ErrNoRows, got %v", err)
}

func TestSetStemmerState_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	state := snowball.Settings{Stemmers: []string{}}
	err := setup.db.setStemmerState(state)
	gta.Assert(t, err == nil)

	fetched, _, err := setup.db.getStemmerState()
	gta.Assert(t, err == nil)

	gta.DeepEqual(t, fetched, state)
}

func TestSetStemmerState_Existing(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	state := snowball.Settings{Stemmers: []string{}}
	err := setup.db.setStemmerState(state)
	gta.Assert(t, err == nil)

	state = snowball.Settings{
		Stemmers: []string{
			"german", "dutch",
		},
		RemoveDiacritics: true,
		TokenCharacters:  "asd",
		Separators:       "zxc",
	}
	err = setup.db.setStemmerState(state)
	gta.Assert(t, err == nil)

	fetched, updated, err := setup.db.getStemmerState()
	gta.Assert(t, err == nil)

	gta.Assert(t, time.Now().After(updated))
	gta.Assert(t, updated.Add(time.Second).After(time.Now()))

	gta.DeepEqual(t, fetched, state)
}
