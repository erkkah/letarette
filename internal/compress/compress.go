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

package compress

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo linux LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo windows LDFLAGS: -L${SRCDIR}/.. -lsqlite
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #cgo CFLAGS: -DMINIZ_NO_STDIO -DMINIZ_NO_ARCHIVE_APIS -DMINIZ_NO_TIME -Dcompress=mz_compress -Duncompress=mz_uncompress
// #include "compress.h"
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mattn/go-sqlite3"
)

// Init registers the spellfix extension with a connection.
func Init(conn *sqlite3.SQLiteConn) error {
	db := dbFromConnection(conn)
	var errorMessage *C.char
	var nullRoutines = (*C.sqlite3_api_routines)(nil)
	result := C.sqlite3_compress_init(db, &errorMessage, nullRoutines)
	if result != C.SQLITE_OK {
		message := C.GoString(errorMessage)
		return fmt.Errorf("failed to init compress extension: %s", message)
	}
	return nil
}

func dbFromConnection(conn *sqlite3.SQLiteConn) *C.sqlite3 {
	dbVal := reflect.ValueOf(conn).Elem().FieldByName("db")
	dbPtr := unsafe.Pointer(dbVal.Pointer())
	return (*C.sqlite3)(dbPtr)
}
