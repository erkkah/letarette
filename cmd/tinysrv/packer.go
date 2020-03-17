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

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Packer compresses strings to save some memory
type Packer struct {
	reader *gzip.Reader
	writer *gzip.Writer
	buffer *bytes.Buffer
}

// NewPacker initializes a packer
func NewPacker() *Packer {
	buffer := new(bytes.Buffer)
	writer := gzip.NewWriter(buffer)
	return &Packer{
		nil,
		writer,
		buffer,
	}
}

// Pack compresses a string to a byte slice
func (p *Packer) Pack(str string) ([]byte, error) {
	p.buffer.Reset()
	p.writer.Reset(p.buffer)
	utf8Bytes := []byte(str)
	written, err := p.writer.Write(utf8Bytes)
	if err != nil {
		return nil, err
	}
	if written != len(utf8Bytes) {
		return nil, fmt.Errorf("unexpected compressor write result")
	}
	err = p.writer.Close()
	if err != nil {
		return nil, err
	}
	packed := p.buffer.Bytes()
	clone := make([]byte, len(packed))
	copy(clone, packed)
	return clone, nil
}

func (p *Packer) resetReader() error {
	if p.reader != nil {
		return p.reader.Reset(p.buffer)
	}
	reader, err := gzip.NewReader(p.buffer)
	p.reader = reader
	return err
}

// Unpack decompresses byte slice to a string
func (p *Packer) Unpack(packed []byte) (string, error) {
	p.buffer.Reset()
	written, err := p.buffer.Write(packed)
	if err != nil {
		return "", err
	}
	if written != len(packed) {
		return "", fmt.Errorf("unexpected buffer write result")
	}
	err = p.resetReader()
	if err != nil {
		return "", err
	}
	unpacked := new(bytes.Buffer)
	_, err = io.Copy(unpacked, p.reader)
	if err != nil {
		return "", err
	}
	return unpacked.String(), nil
}
