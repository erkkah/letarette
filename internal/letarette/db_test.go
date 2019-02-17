package letarette

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

type testSetup struct {
	tmpDir string
	config Config
}

func (setup *testSetup) cleanup() {
	err := os.RemoveAll(setup.tmpDir)
	if err != nil {
		panic("Failed to delete test temp dir")
	}
}

func getTestSetup() *testSetup {
	setup := new(testSetup)
	var err error
	setup.tmpDir, err = ioutil.TempDir("", "letarette")
	if err != nil {
		panic("Failed to create test temp dir")
	}
	setup.config.Db.Path = path.Join(setup.tmpDir, "leta.db")
	setup.config.Index.Spaces = []string{"test"}
	return setup
}

func TestOpen(t *testing.T) {
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Errorf("Database is nil!")
	}
}

func TestGetLastUpdateTime_ExistingSpace(t *testing.T) {
	then := time.Unix(1, 0)
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	last, err := db.getLastUpdateTime("test")
	if err != nil {
		t.Errorf("Failed to get last update time: %v", err)
	}
	if !last.Before(then) {
		t.Errorf("Initial update time should be before %v, got %v", then, last)
	}
}

func TestGetLastUpdateTime_NonExistingSpace(t *testing.T) {
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.getLastUpdateTime("popowkqd")
	if err == nil {
		t.Errorf("Fetching last update time for unknown space should fail!")
	}
}

func TestSetLastUpdateTime_NonExistingSpace(t *testing.T) {
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.setLastUpdateTime("popowkqd", time.Now())
	if err == nil {
		t.Errorf("Setting last update time for unknown space should fail!")
	}
}

func TestSetLastUpdateTime_ExistingSpace(t *testing.T) {
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	theTime := time.Now()
	err = db.setLastUpdateTime("test", theTime)
	if err != nil {
		t.Errorf("Failed to set last update time: %v", err)
	}

	readTime, err := db.getLastUpdateTime("test")
	if err != nil {
		t.Errorf("Failed to read back last update time: %v", err)
	}
	if !readTime.Equal(theTime) {
		t.Errorf("Read back time value differs (%v != %v)", theTime, readTime)
	}
}

func TestGetInterestList_Empty(t *testing.T) {
	setup := getTestSetup()
	defer setup.cleanup()

	db, err := OpenDatabase(setup.config)
	if err != nil {
		t.Errorf("Failed to open database: %v", err)
	}
	defer db.Close()

	list, err := db.getInterestList("test")
	if err != nil {
		t.Errorf("Failed to get interest list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Length should be empty")
	}
}
