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

func TestSetLastUpdateTime_NonExistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	err := setup.db.setLastUpdateTime("popowkqd", time.Now())
	if err == nil {
		t.Errorf("Setting last update time for unknown space should fail!")
	}
}

func TestSetLastUpdateTime_ExistingSpace(t *testing.T) {
	setup := getTestSetup(t)
	defer setup.cleanup()

	theTime := time.Now()
	err := setup.db.setLastUpdateTime("test", theTime)
	if err != nil {
		t.Errorf("Failed to set last update time: %v", err)
	}

	readTime, err := setup.db.getLastUpdateTime("test")
	if err != nil {
		t.Errorf("Failed to read back last update time: %v", err)
	}
	if !readTime.Equal(theTime) {
		t.Errorf("Read back time value differs (%v != %v)", theTime, readTime)
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
