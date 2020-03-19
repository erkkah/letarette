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

package client

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

func connect(URLs []string, opts state) (*nats.EncodedConn, error) {
	natsOptions := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
	}

	if len(opts.rootCAs) > 0 {
		natsOptions = append(natsOptions, nats.RootCAs(opts.rootCAs...))
	}

	if opts.seedFile != "" {
		option, err := nats.NkeyOptionFromSeed(opts.seedFile)
		if err != nil {
			return nil, err
		}
		natsOptions = append(natsOptions, option)
	}

	nc, err := nats.Connect(strings.Join(URLs, ","), natsOptions...)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

const COMPRESSED_ENCODER = "COMPRESSED_ENCODER"
const COMPRESSION_MARKER = uint8(0xf8)
const COMPRESSION_LIMIT = 1024

type CompressedJSONEncoder struct{}

func (e CompressedJSONEncoder) Encode(subject string, value interface{}) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	if len(encoded) <= COMPRESSION_LIMIT {
		return encoded, nil
	}

	var buf bytes.Buffer
	_ = buf.WriteByte(COMPRESSION_MARKER)

	writer, err := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	if err != nil {
		return nil, err
	}
	_, err = writer.Write(encoded)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	return buf.Bytes(), err
}

func (e CompressedJSONEncoder) Decode(subject string, data []byte, valuePointer interface{}) error {
	byteReader := bytes.NewReader(data)
	marker, err := byteReader.ReadByte()
	if err != nil {
		return err
	}

	var bytes []byte

	if marker != COMPRESSION_MARKER {
		_ = byteReader.UnreadByte()
		bytes, _ = ioutil.ReadAll(byteReader)
	} else {
		zReader, err := zlib.NewReader(byteReader)
		if err != nil {
			return err
		}
		bytes, err = ioutil.ReadAll(zReader)
		if err != nil {
			return err
		}
	}
	return json.Unmarshal(bytes, valuePointer)
}

func init() {
	nats.RegisterEncoder(COMPRESSED_ENCODER, CompressedJSONEncoder{})
}
