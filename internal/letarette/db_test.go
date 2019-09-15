package letarette

import (
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"
	"time"
)

type testSetup struct {
	tmpDir string
	config Config
	db     Database
}

func (setup *testSetup) cleanup() {
	err := os.RemoveAll(setup.tmpDir)
	if err != nil {
		panic("Failed to delete test temp dir")
	}
	if setup.db != nil {
		setup.db.Close()
	}
}

func getTestSetup(t *testing.T) *testSetup {
	setup := new(testSetup)
	var err error
	setup.tmpDir, err = ioutil.TempDir("", "letarette")
	if err != nil {
		t.Fatal("Failed to create test temp dir")
	}
	setup.config.Db.Path = path.Join(setup.tmpDir, "leta.db")
	setup.config.Index.Spaces = []string{"test"}

	setup.db, err = OpenDatabase(setup.config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	return setup
}

func TestOpen(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	if setup.db == nil {
		t.Errorf("Database is nil!")
	}
}

func TestAddDocument_EmptyDocument(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	doc := Document{}
	err := setup.db.addDocumentUpdate(doc)
	if err == nil {
		t.Errorf("Adding empty document should fail")
	}
}

func TestAddDocument_NewDocument(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	doc := Document{
		Space:   "test",
		ID:      "myID",
		Updated: time.Now(),
		Text:    "tjo och hej",
		Alive:   true,
	}
	err := setup.db.addDocumentUpdate(doc)
	if err != nil {
		t.Errorf("Failed to add new document: %v", err)
	}
}

func TestCommitInterestList_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	err := setup.db.commitInterestList("test")
	if err != nil {
		t.Errorf("Failed to commit empty list: %v", err)
	}
}

func TestCommitInterestList_NonEmptyNoUpdates(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	beforeState, err := setup.db.getInterestListState("test")
	if err != nil {
		t.Errorf("Failed to get list state: %v", err)
	}

	list := []DocumentID{"bello", "koko"}

	err = setup.db.setInterestList("test", list)
	if err != nil {
		t.Errorf("Setting interest list failed: %v", err)
	}

	err = setup.db.commitInterestList("test")
	if err != nil {
		t.Errorf("Failed to commit list: %v", err)
	}

	afterState, err := setup.db.getInterestListState("test")
	if err != nil {
		t.Errorf("Failed to get list state: %v", err)
	}

	if beforeState.LastUpdated != afterState.LastUpdated {
		t.Errorf("Expected untouched state")
	}
	if beforeState.LastUpdatedDocID != afterState.LastUpdatedDocID {
		t.Errorf("Expected untouched state")
	}
}

func TestCommitInterestList_NonEmptyWithUpdates(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := []DocumentID{"bello", "koko"}
	docTime := time.Now()
	docID := DocumentID("koko")
	doc := Document{
		Space:   "test",
		ID:      docID,
		Updated: docTime,
		Text:    "tjo och hej",
		Alive:   true,
	}

	err := setup.db.setInterestList("test", list)
	if err != nil {
		t.Errorf("Setting interest list failed: %v", err)
	}

	err = setup.db.addDocumentUpdate(doc)
	if err != nil {
		t.Errorf("Failed to add document: %v", err)
	}

	err = setup.db.commitInterestList("test")
	if err != nil {
		t.Errorf("Failed to commit list: %v", err)
	}

	afterState, err := setup.db.getInterestListState("test")
	if err != nil {
		t.Errorf("Failed to get list state: %v", err)
	}

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

	last, err := setup.db.getLastUpdateTime("test")
	if err != nil {
		t.Errorf("Failed to get last update time: %v", err)
	}
	if !last.Before(then) {
		t.Errorf("Initial update time should be before %v, got %v", then, last)
	}
}

func TestGetLastUpdateTime_NonExistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	_, err := setup.db.getLastUpdateTime("popowkqd")
	if err == nil {
		t.Errorf("Fetching last update time for unknown space should fail!")
	}
}

func TestGetInterestList_Empty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list, err := setup.db.getInterestList("test")
	if err != nil {
		t.Errorf("Failed to get interest list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Length should be empty")
	}
}

func TestGetInterestList_NonexistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	_, err := setup.db.getInterestList("kawonka")
	if err == nil {
		t.Errorf("Fetching interest list for nonexisting space should fail!")
	}
}

func TestSetInterestList_NonexistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	err := setup.db.setInterestList("kawonka", []DocumentID{"koko"})
	if err == nil {
		t.Errorf("Setting interest list for nonexisting space should fail!")
	}
}

func TestSetGetInterestList_CurrentListEmpty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := []DocumentID{"bello", "koko"}

	err := setup.db.setInterestList("test", list)
	if err != nil {
		t.Errorf("Setting interest list failed: %v", err)
	}

	fetchedSlice, err := setup.db.getInterestList("test")
	if err != nil {
		t.Errorf("Getting interest list failed: %v", err)
	}
	sort.Slice(fetchedSlice, func(i int, j int) bool {
		return fetchedSlice[i].DocID < fetchedSlice[j].DocID
	})

	for i, interest := range fetchedSlice {
		if interest.Served {
			t.Errorf("New interest should be unserved")
		}
		if interest.DocID != list[i] {
			t.Errorf("New list does not match")
		}
	}
}

func TestSetInterestList_CurrentListNonEmpty(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	list := [2]DocumentID{"bello", "koko"}

	err := setup.db.setInterestList("test", list[:])
	if err != nil {
		t.Errorf("Setting interest list failed: %v", err)
	}

	err = setup.db.setInterestList("test", list[:])
	if err == nil {
		t.Errorf("Setting interest list with current list should fail!")
	}
}
