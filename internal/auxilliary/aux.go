package auxilliary

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo darwin CFLAGS: -I/usr/local/opt/sqlite/include
// #cgo LDFLAGS: -lsqlite3
// #cgo darwin LDFLAGS: -L/usr/local/opt/sqlite/lib
// #include "aux.h"
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mattn/go-sqlite3"
)

func Init(conn *sqlite3.SQLiteConn) error {
	db := dbFromConnection(conn)
	result := C.initAuxilliaryFunctions(db)
	if result != C.SQLITE_OK {
		return fmt.Errorf("Failed to init auxilliary functions")
	}
	return nil
}

func dbFromConnection(conn *sqlite3.SQLiteConn) *C.sqlite3 {
	dbVal := reflect.ValueOf(conn).Elem().FieldByName("db")
	dbPtr := unsafe.Pointer(dbVal.Pointer())
	return (*C.sqlite3)(dbPtr)
}
