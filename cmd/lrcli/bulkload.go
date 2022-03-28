// Copyright 2020 Erik Agsjö
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

package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/erkkah/letarette/pkg/spinner"
)

type entry struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Text  string    `json:"text"`
	Date  time.Time `json:"date"`
}

type bulkLoadOptions struct {
	databaseOptions
	Space      string `arg:"0"`
	JSON       string `arg:"1"`
	AutoAssign bool   `name:"a"`
	LoadLimit  int    `name:"m"`
}

func bulkLoad(db letarette.Database, options bulkLoadOptions, shardGroupSize int, shardIndex int) {
	s := spinner.New(os.Stdout)
	s.Start("Loading ")

	start := time.Now()

	objFile := options.JSON
	var fileReader io.Reader

	file, err := os.Open(objFile)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to open file: %v", err))
		return
	}
	defer file.Close()
	fileReader = file

	if strings.HasSuffix(objFile, ".gz") {
		gzipReader, err := gzip.NewReader(fileReader)
		if err != nil {
			s.Stop(fmt.Sprintf("Failed to open gzipped file: %v", err))
			return
		}
		defer gzipReader.Close()
		fileReader = gzipReader
	}

	decoder := json.NewDecoder(fileReader)

	numLoaded := 0

	loader, err := letarette.StartBulkLoad(db, options.Space)
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to init bulkloader: %v", err))
		return
	}
	defer func() {
		if loader != nil {
			_ = loader.Rollback()
		}
	}()

	numID := 0
	epoch := time.Unix(0, 0)

	for options.LoadLimit == 0 || numLoaded < options.LoadLimit {
		var e entry
		readErr := decoder.Decode(&e)

		if readErr == nil {
			if options.AutoAssign {
				e.ID = strconv.Itoa(numID)
				numID++
			}
			if e.ID == "" {
				s.Stop("Cannot load document without ID, use -a for auto-assign?\n")
				return
			}
			index := letarette.ShardIndexFromDocumentID(protocol.DocumentID(e.ID), shardGroupSize)
			if index != shardIndex {
				logger.Debug.Printf("Skipping document %q, not for the configured shard", e.ID)
				continue
			}

			logger.Debug.Printf("Loading document %q", e.ID)

			if e.Date.Before(epoch) {
				logger.Info.Printf("Resetting invalid date to epoch")
				e.Date = epoch
			}
			doc := protocol.Document{
				ID:      protocol.DocumentID(e.ID),
				Title:   e.Title,
				Text:    e.Text,
				Alive:   true,
				Updated: e.Date,
			}
			err = loader.Load(doc)
			if err != nil {
				s.Stop(fmt.Sprintf("Document load error: %v\n", err))
				return
			}
			numLoaded++
		} else {
			if !errors.Is(readErr, io.EOF) {
				s.Stop(fmt.Sprintf("Read error: %v\n", readErr))
				return
			}
			break
		}
	}

	err = loader.Commit()
	if err != nil {
		s.Stop(fmt.Sprintf("Failed to commit after bulk load: %v\n", err))
		return
	}

	elapsed := time.Since(start)
	loadedMegs := float64(loader.LoadedBytes()) / 1000 / 1000
	performance := loadedMegs / elapsed.Seconds()

	s.Stop(fmt.Sprintf("Loaded %v documents in %v, %.2f Mbytes, %.2f Mbytes/s\n",
		numLoaded, elapsed, loadedMegs, performance))
	loader = nil
}
