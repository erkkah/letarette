package spellfix

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo darwin CFLAGS: -I/usr/local/opt/sqlite/include
// #cgo LDFLAGS: -lsqlite3
// #cgo darwin LDFLAGS: -L/usr/local/opt/sqlite/lib
// #include "spellfix.h"
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mattn/go-sqlite3"
)

func Init(conn *sqlite3.SQLiteConn) error {
	db := dbFromConnection(conn)
	var errorMessage *C.char
	var nullRoutines = (*C.sqlite3_api_routines)(nil)
	result := C.sqlite3_spellfix_init(db, &errorMessage, nullRoutines)
	if result != C.SQLITE_OK {
		message := C.GoString(errorMessage)
		return fmt.Errorf("Failed to init spellfix extension: %s", message)
	}
	return nil
}

func dbFromConnection(conn *sqlite3.SQLiteConn) *C.sqlite3 {
	dbVal := reflect.ValueOf(conn).Elem().FieldByName("db")
	dbPtr := unsafe.Pointer(dbVal.Pointer())
	return (*C.sqlite3)(dbPtr)
}
