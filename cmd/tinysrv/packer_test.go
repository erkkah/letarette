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
	"encoding/base64"
	"math/rand"
	"testing"

	gta "gotest.tools/assert"
)

func TestCreatePacker(t *testing.T) {
	p := NewPacker()
	gta.Assert(t, p != nil)
}

func TestPackAndUnpack_OneShort(t *testing.T) {
	p := NewPacker()
	const msg = "Tjillevippen, plippen!"
	packed, err := p.Pack(msg)
	gta.NilError(t, err)
	unpacked, err := p.Unpack(packed)
	gta.NilError(t, err)
	gta.Equal(t, msg, unpacked)
}

func randomString(length int) string {
	randomBytes := make([]byte, length)
	rand.Read(randomBytes)
	var encoded bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &encoded)
	encoder.Write(randomBytes)
	encoder.Close()
	return encoded.String()
}

func TestPackAndUnpack_SeveralShort(t *testing.T) {
	p := NewPacker()

	for i := 0; i < 10; i++ {
		msg := randomString(512)
		packed, err := p.Pack(msg)
		gta.NilError(t, err)
		unpacked, err := p.Unpack(packed)
		gta.NilError(t, err)
		gta.Equal(t, msg, unpacked)
	}
}

func TestPackAndUnpack_Long(t *testing.T) {
	p := NewPacker()

	msg := randomString(256 * 1024)
	packed, err := p.Pack(msg)
	gta.NilError(t, err)
	unpacked, err := p.Unpack(packed)
	gta.NilError(t, err)
	gta.Equal(t, msg, unpacked)
}
