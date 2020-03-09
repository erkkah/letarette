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

package snowball

import (
	"fmt"
	"reflect"
	"unsafe"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo CFLAGS: -Iext/libstemmer_c/include
// #cgo LDFLAGS: ${SRCDIR}/ext/libstemmer_c/libstemmer.o
// #cgo linux LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #cgo dbstats CFLAGS: -DSQLITE_ENABLE_DBSTAT_VTAB=1
// #include "snowball.h"
// #include <stdlib.h>
import "C"

// Settings for initializing the stemmer
type Settings struct {
	Stemmers         []string
	RemoveDiacritics bool
	TokenCharacters  string
	Separators       string
	MinTokenLength   int
}

// ListStemmers returns a list of all built-in Snowball
// stemmer algorithms.
func ListStemmers() []string {
	var cStemmers = C.getStemmerList()
	var stemmerArray = (*[100](*C.char))(unsafe.Pointer(cStemmers))
	var stemmers = []string{}
	for i := 0; stemmerArray[i] != nil; i++ {
		stemmers = append(stemmers, C.GoString(stemmerArray[i]))
	}
	return stemmers
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

	var removeDiacritics = 0
	if settings.RemoveDiacritics {
		/*
			??? Does not seem to work on OSX?
			if C.SQLITE_VERSION_NUMBER < 3027001 {
				cRemoveDiacritics = 1
			} else {
				cRemoveDiacritics = 2
			}
		*/
		removeDiacritics = 1
	}

	minTokenLength := 2
	if settings.MinTokenLength > 0 {
		minTokenLength = settings.MinTokenLength
	}

	result := C.initSnowballStemmer(
		db,
		cStemmers, C.int(len(settings.Stemmers)),
		C.int(removeDiacritics), cTokenCharacters, cSeparators,
		C.int(minTokenLength),
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
