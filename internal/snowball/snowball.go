package snowball

//go:generate make -C snowball

import (
	"fmt"
	"reflect"
	"unsafe"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// #cgo CFLAGS: -DSQLITE_CORE -Isnowball/include
// #cgo LDFLAGS: -lsqlite3 ${SRCDIR}/snowball/libstemmer.o
// #include "snowball.h"
// #include <stdlib.h>
import "C"

// Settings for initializing the stemmer
type Settings struct {
	Stemmers         []string
	RemoveDiacritics bool
	TokenCharacters  string
	Separators       string
}

// Init registers the snowball stemmer with the connection and configures
// it for the list of languages.
// If a language cannot be found, initialization fails.
func Init(conn *sqlite3.SQLiteConn, settings Settings) error {
	if len(settings.Stemmers) == 0 {
		return fmt.Errorf("config.Stemmers list cannot be empty")
	}

	db := dbFromConnection(conn)
	cStemmers := allocateCArgs(settings.Stemmers)

	var cTokenCharacters *C.char
	if len(settings.TokenCharacters) > 0 {
		cTokenCharacters = C.CString(settings.TokenCharacters)
	}

	var cSeparators *C.char
	if len(settings.Separators) > 0 {
		cSeparators = C.CString(settings.Separators)
	}

	var cRemoveDiacritics = 0
	if settings.RemoveDiacritics {
		if C.SQLITE_VERSION_NUMBER < 3027001 {
			cRemoveDiacritics = 1
		} else {
			cRemoveDiacritics = 2
		}
	}
	result := C.initSnowballStemmer(
		db, cStemmers, C.int(len(settings.Stemmers)), C.int(cRemoveDiacritics), cTokenCharacters, cSeparators,
	)

	freeCArgs(cStemmers, len(settings.Stemmers))

	if cSeparators != nil {
		C.free(unsafe.Pointer(cSeparators))
	}

	if cTokenCharacters != nil {
		C.free(unsafe.Pointer(cTokenCharacters))
	}

	if result != C.SQLITE_OK {
		return fmt.Errorf("Failed to init snowball, check language list")
	}
	return nil
}

func dbFromConnection(conn *sqlite3.SQLiteConn) *C.sqlite3 {
	dbVal := reflect.ValueOf(conn).Elem().FieldByName("db")
	dbPtr := unsafe.Pointer(dbVal.Pointer())
	return (*C.sqlite3)(dbPtr)
}

const maxArgs = 512

func allocateCArgs(args []string) **C.char {
	if len(args) > maxArgs {
		panic("Argument array > 512 items")
	}
	cArgs := C.malloc(C.size_t(len(args)) * C.size_t(unsafe.Sizeof(uintptr(0))))

	a := (*[maxArgs]*C.char)(cArgs)
	for i, v := range args {
		a[i] = C.CString(v)
	}

	return (**C.char)(cArgs)
}

func freeCArgs(cArgs **C.char, nArgs int) {
	a := (*[maxArgs]*C.char)(unsafe.Pointer(cArgs))
	for i := 0; i < nArgs; i++ {
		C.free(unsafe.Pointer(a[i]))
	}
	C.free(unsafe.Pointer(cArgs))
}