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

package letarette

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	lrio "github.com/erkkah/letarette/pkg/io"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/jmoiron/sqlx"
)

// A ShardCloner creates a copy of all documents in the index that
// matches a specific shard group. The result is a gzipped, gob-encoded
// file, ready to be loaded.
type ShardCloner struct {
	encoder      *gob.Encoder
	compressor   *gzip.Writer
	dest         io.Writer
	rows         *sqlx.Rows
	docStatement *sqlx.Stmt
	targetIndex  int
	targetSize   int
	count        int
}

type cloneDocument struct {
	protocol.Document
	RowID        int64 `db:"rowid"`
	Space        string
	UpdatedNanos int64 `db:"updatedNanos"`
}

const (
	currentCloneVersion = 1
)

// StartShardClone starts the process of cloning all documents in the index for loading
// into a specified shard group.
func StartShardClone(ctx context.Context, db *database, shardGroup string, dest io.Writer) (*ShardCloner, error) {
	group, size, err := parseShardString(shardGroup)
	if err != nil {
		return nil, err
	}

	rowQ := "select docs.id as rowid, docID as id, space from docs join spaces using (spaceID)"

	rows, err := db.rdb.QueryxContext(ctx, rowQ)
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	statement, err := db.rdb.PreparexContext(
		ctx, `select updatedNanos, title, txt as "text", alive from docs where id = ?`,
	)

	if err != nil {
		return nil, err
	}

	compressor := gzip.NewWriter(dest)
	encoder := gob.NewEncoder(compressor)

	var version int = currentCloneVersion
	err = encoder.Encode(version)
	if err != nil {
		return nil, err
	}

	return &ShardCloner{
		dest:         dest,
		compressor:   compressor,
		encoder:      encoder,
		rows:         rows,
		docStatement: statement,
		targetIndex:  group - 1,
		targetSize:   size,
	}, nil
}

const docsPerCloneStep = 1000

// Step runs one step of the cloning process.
// Returns false when cloning is complete.
func (s *ShardCloner) Step(ctx context.Context) (bool, error) {
	var doc cloneDocument
	var currentSpace string

	for docs := 0; docs < docsPerCloneStep; docs++ {
		select {
		case <-ctx.Done():
			return true, fmt.Errorf("interrupted")
		default:
		}
		if !s.rows.Next() {
			return false, nil
		}
		err := s.rows.StructScan(&doc)
		if err != nil {
			return true, err
		}
		index := ShardIndexFromDocumentID(doc.ID, s.targetSize)
		if index != s.targetIndex {
			continue
		}

		rows, err := s.docStatement.QueryxContext(ctx, doc.RowID)
		if err != nil {
			return true, err
		}
		if rows.Err() != nil {
			return true, rows.Err()
		}
		rows.Next()
		err = rows.StructScan(&doc)
		if err != nil {
			return true, err
		}
		_ = rows.Close()

		doc.Updated = time.Unix(0, doc.UpdatedNanos)

		if doc.Space != currentSpace {
			currentSpace = doc.Space
		} else {
			doc.Space = ""
		}
		err = s.encoder.Encode(doc.Space)
		if err != nil {
			return true, err
		}

		err = s.encoder.Encode(doc.Document)
		if err != nil {
			return true, err
		}

		s.count++
	}
	return true, nil
}

// Close stops the cloning process and closes the output.
func (s *ShardCloner) Close() (int, error) {
	_ = s.rows.Close()
	_ = s.docStatement.Close()

	err := s.compressor.Close()
	if err != nil {
		return s.count, err
	}

	return s.count, nil
}

// LoadShardClone loads a clone from a reader source
func LoadShardClone(ctx context.Context, db Database, source io.Reader) error {
	uncompressor, err := gzip.NewReader(lrio.NewReaderContext(ctx, source))
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(uncompressor)

	var version int
	err = decoder.Decode(&version)
	if err != nil {
		return err
	}
	if version > currentCloneVersion {
		return fmt.Errorf("incompatible clone format")
	}

	var loader *BulkLoader
	defer func() {
		if loader != nil {
			_ = loader.Rollback()
		}
	}()

	for {

		var space string
		err = decoder.Decode(&space)

		if err != nil {
			if err == io.EOF {
				if loader != nil {
					err = loader.Commit()
					loader = nil
					return err
				}
				return nil
			}
			return fmt.Errorf("failed to read space: %w", err)
		}

		if space != "" {
			if loader != nil {
				err = loader.Commit()
				if err != nil {
					return err
				}
			}
			loader, err = StartBulkLoad(db, space)
			if err != nil {
				return fmt.Errorf("failed to start bulk load: %w", err)
			}
		}

		if loader == nil {
			return fmt.Errorf("unexpected clone format")
		}

		var doc protocol.Document
		err = decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				err = loader.Commit()
				loader = nil
				return err
			}
			return fmt.Errorf("failed to read doc: %w", err)
		}

		err = loader.Load(doc)
		if err != nil {
			return err
		}
	}
}
