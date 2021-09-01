// Copyright 2020 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/pkg/spinner"
)

func loadSynonyms(db letarette.Database) {
	s := spinner.New(os.Stdout)
	s.Start("Loading ")
	defer s.Stop()

	objFile := cmdline.JSON
	var fileReader io.Reader

	file, err := os.Open(objFile)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to open file: %v\n", err))
		return
	}
	defer file.Close()
	fileReader = file

	if strings.HasSuffix(objFile, ".gz") {
		gzipReader, err := gzip.NewReader(fileReader)
		if err != nil {
			s.Stop(fmt.Sprintf("Failed to open gzipped file: %v\n", err))
			return
		}
		defer gzipReader.Close()
		fileReader = gzipReader
	}

	decoder := json.NewDecoder(fileReader)
	var voidSynonym []interface{}
	var synonyms []letarette.Synonyms

	count := 0

	for {
		err := decoder.Decode(&voidSynonym)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.Stop(fmt.Sprintf("Read error: %v\n", err))
				return
			}
			break
		}
		count++
		description := voidSynonym[0].(string)
		voidSynonyms := voidSynonym[1].([]interface{})
		var synonymWords []string
		for _, r := range voidSynonyms {
			synonymWords = append(synonymWords, r.(string))
		}
		synonyms = append(synonyms, letarette.Synonyms{
			Description: description,
			Words:       synonymWords,
		})
	}

	ctx := context.Background()
	err = letarette.SetSynonyms(ctx, db, synonyms)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to update synonyms: %v\n", err))
		return
	}

	s.Stop(strconv.Itoa(count), "synonym groups loaded.\n")
}

func dumpSynonyms(db letarette.Database) {
	lines, err := db.RawQuery(`
	with syn as (
		select
			description, json_group_array(word) as syns
		from
			synonyms s join synonym_words sw on s.id = sw.synonymID
		group by
			s.id
	)
	select
		json_array(description, syns)
	from syn	
	`)
	if err != nil {
		fmt.Printf("Failed to dump synonyms: %v", err)
		return
	}
	for _, line := range lines {
		fmt.Printf("%v\n", line)
	}
}
