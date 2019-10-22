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
	return string(encoded.Bytes())
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
