// Code generated for package letarette by go-bindata DO NOT EDIT. (@generated)
// sources:
// migrations/1_init.down.sql
// migrations/1_init.up.sql
// migrations/2_alive.down.sql
// migrations/2_alive.up.sql
// migrations/3_snowball.down.sql
// migrations/3_snowball.up.sql
// migrations/4_stemmerstate.down.sql
// migrations/4_stemmerstate.up.sql
// migrations/5_spaceindex.down.sql
// migrations/5_spaceindex.up.sql
// migrations/6_prefixindex.down.sql
// migrations/6_prefixindex.up.sql
// migrations/7_interesttime.down.sql
// migrations/7_interesttime.up.sql
// migrations/8_indexid.down.sql
// migrations/8_indexid.up.sql
// queries/search_1.sql
// queries/search_2.sql
package letarette

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _migrations1_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\x29\xca\x2f\x50\x28\x49\x4c\xca\x49\x55\x28\x2e\x48\x4c\x4e\x2d\xb6\xe6\xe2\x42\x12\xcb\xcc\x2b\x49\x2d\x4a\x2d\x2e\x41\x15\x4d\xc9\x4f\x46\x53\x97\x56\x52\x6c\xcd\x05\x08\x00\x00\xff\xff\x40\xc9\x79\xc7\x4c\x00\x00\x00")

func migrations1_initDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations1_initDownSql,
		"migrations/1_init.down.sql",
	)
}

func migrations1_initDownSql() (*asset, error) {
	bytes, err := migrations1_initDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/1_init.down.sql", size: 76, mode: os.FileMode(436), modTime: time.Unix(1570371832, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations1_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x93\x3f\x6f\xdb\x30\x10\xc5\x77\x7d\x8a\x83\x3b\x48\x02\x94\xa2\x1d\xda\x25\xf5\x50\xd4\x8b\x97\x6e\x9d\x03\x86\x3c\x29\x07\xd3\x47\x95\x3c\x26\x4e\x3f\x7d\x41\x52\x72\x9b\x44\xce\x1f\x04\x9e\x24\xfb\xc8\x77\xbf\x7b\xef\xa4\x3d\x2a\x41\x10\x75\x6d\x11\xa8\x07\x76\x02\x78\xa0\x20\x01\xc2\xa8\x34\x86\xa6\x02\x80\xf2\xbe\xdd\x00\xb1\xe0\x80\x1e\x46\x4f\x7b\xe5\xef\x61\x87\xf7\xdd\xbf\x03\x20\x78\x90\x2c\xc1\xd1\x5a\x88\x4c\xbf\x23\x76\x55\x3e\x70\x71\x01\xc4\x06\x0f\x30\xba\x40\x42\x8e\x41\x68\x8f\x41\xd4\x7e\xcc\x75\xab\x82\xfc\x1a\x8d\x12\x34\xdf\xe5\xa7\x62\x17\x8e\xcd\x66\xc1\xee\x84\x90\x71\x3a\xee\x91\x65\xbb\x79\xac\xb4\x71\x7a\xbb\x79\x44\x65\xb0\x57\xd1\x0a\xac\x56\xff\x93\x09\x7a\x0c\x02\x96\x82\x40\xf6\x64\x81\x90\x82\xfc\xc8\x76\x9d\x24\x3c\x8a\x7f\x9a\xb4\xf5\x0d\xea\x5d\xb1\xf0\xc4\x90\xdf\xd6\x0b\xc2\xf9\x42\x5b\xb5\x97\x55\xf5\x4c\x40\x33\xf5\x72\x44\x0f\x5d\x33\x4f\x9d\x98\x82\x93\x24\xbf\x7c\xa9\x04\xd8\x4c\xca\x5d\x11\x69\x73\xa9\x77\x1e\x69\xe0\xb4\x00\x30\x1f\x68\xc1\x63\x8f\x1e\x59\xe3\x71\x7b\xe6\xd2\x4b\xb3\x18\xa7\xa7\x55\x23\xf3\xc2\x96\xbd\x7d\xc6\x58\x2c\x7f\x6e\xab\xe4\x20\x8b\x37\xcf\xe3\xc0\x2d\x79\x89\xca\x2e\x3a\xd1\x4b\x80\x18\x88\x87\xf4\xf6\xa5\x99\xe9\x3a\xd0\x8e\x05\x59\xd6\x75\xf2\xaa\x3e\xfe\xbe\xf2\xee\x8e\xcc\xba\x26\x53\x4f\xa3\xb8\x1d\x32\xfd\xc1\xf5\x6a\x74\x5e\xd0\xa7\x21\xb4\x33\xf8\xf5\x73\x29\xe9\x1b\xe5\x03\xd4\x1f\xea\xd5\x83\x50\x3c\x0d\xc9\x96\xa7\xb1\x5c\x29\x02\xd5\x27\x21\xe2\x80\x5e\xa0\x7c\x72\x01\xae\x71\x20\x2e\x99\x95\x02\xb1\xb8\x44\xdd\x64\xa4\x2e\x61\xb7\x70\xab\x6c\xc4\x00\x0d\xe3\xdd\xc7\xf4\x67\x7a\xa6\xc2\x65\x85\x6c\x5e\xd7\xdf\x4c\xfd\x0d\x5a\x14\x7c\x45\xff\x5e\x42\x07\x4b\x10\x75\x91\xa8\x3b\x70\xd6\x64\x9c\xf4\x7c\x23\x4e\x9c\x70\xca\x5a\x9d\x0b\xe7\x9d\xbe\xfe\x0d\x00\x00\xff\xff\x1b\xf2\x2f\x45\xd9\x05\x00\x00")

func migrations1_initUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations1_initUpSql,
		"migrations/1_init.up.sql",
	)
}

func migrations1_initUpSql() (*asset, error) {
	bytes, err := migrations1_initUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/1_init.up.sql", size: 1497, mode: os.FileMode(436), modTime: time.Unix(1572185519, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations2_aliveDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x04\xc0\x51\x0a\x02\x20\x0c\x06\xe0\xf7\x4e\xf1\x23\xc8\x0a\xa2\x0b\x74\x98\x58\xfa\x0b\xc2\x74\xa0\xab\x5e\x3c\x7c\x9f\x5a\x70\x21\xf4\x6d\x44\xf5\xb2\xb1\x38\x75\x10\xc5\xed\x33\x26\xd4\xfa\x97\x08\x47\xaa\x34\x06\xeb\x2b\xe1\x1c\xec\x58\x2d\xfa\xe0\x55\xf2\x7e\xe4\x26\x77\x99\xfe\x93\xdb\xf3\xf2\x0f\x00\x00\xff\xff\xcf\xb9\xfa\x8c\x4f\x00\x00\x00")

func migrations2_aliveDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations2_aliveDownSql,
		"migrations/2_alive.down.sql",
	)
}

func migrations2_aliveDownSql() (*asset, error) {
	bytes, err := migrations2_aliveDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/2_alive.down.sql", size: 79, mode: os.FileMode(436), modTime: time.Unix(1570372608, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations2_aliveUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x04\xc0\xd1\x09\xc3\x40\x0c\x03\xd0\xff\x4e\xa1\x3d\x3a\x8d\xee\xac\x42\x41\xb1\xe1\x62\x67\xfe\x3c\xba\x75\xd0\x5c\x16\xa2\xf6\x0d\x46\x60\x97\xe7\x4a\xd0\xff\x47\x58\x55\x16\x13\x59\x8d\x1c\x1b\xa1\x1f\xc7\x8d\x3e\xa3\xef\xe7\x0d\x00\x00\xff\xff\x4e\xaf\xf6\xd8\x41\x00\x00\x00")

func migrations2_aliveUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations2_aliveUpSql,
		"migrations/2_alive.up.sql",
	)
}

func migrations2_aliveUpSql() (*asset, error) {
	bytes, err := migrations2_aliveUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/2_alive.up.sql", size: 65, mode: os.FileMode(436), modTime: time.Unix(1570371967, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations3_snowballDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\x8d\xc1\x4a\xc5\x40\x0c\x45\xf7\xf3\x15\xe1\xb9\x48\x0b\xdd\xb8\xd0\x4d\xe9\xb7\xc8\x74\x26\xd5\xe0\x90\x94\x24\x53\x8b\x5f\x2f\xa5\xf2\x76\xf7\x72\x0e\x9c\x6a\xba\x43\xe4\xb5\x11\x6c\xe1\x73\x4a\xc5\x28\x07\xc1\xc1\x16\x3d\xb7\x7f\xc4\x1b\x88\x06\xd0\xc9\x1e\x7e\x89\xd0\x9d\xe5\xf3\x5a\x6f\x43\x02\x00\x88\x33\x26\x28\x2a\x41\x12\x0b\x56\x2d\x8e\xcf\xff\x61\xfa\xc3\x75\x41\xae\x38\xdd\xb2\x7e\x93\xf0\x2f\x2d\x8f\x5d\x2d\xc8\xa0\x0b\x17\xad\xf4\xfe\x7a\xa3\xf2\x95\xcd\x01\x5f\xf0\x91\xc6\x39\x25\x16\x27\x0b\x60\x09\xbd\x8a\xc3\x16\x3e\xc2\x91\x5b\x27\x1f\xd0\x68\xed\xdc\x2a\x8e\x73\xfa\x0b\x00\x00\xff\xff\xfe\x51\x31\xfe\xcc\x00\x00\x00")

func migrations3_snowballDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations3_snowballDownSql,
		"migrations/3_snowball.down.sql",
	)
}

func migrations3_snowballDownSql() (*asset, error) {
	bytes, err := migrations3_snowballDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/3_snowball.down.sql", size: 204, mode: os.FileMode(436), modTime: time.Unix(1571435094, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations3_snowballUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\xcc\xc1\x0a\x83\x30\x10\x04\xd0\x7b\xbe\x62\xf1\x12\x05\xaf\x3d\x89\xdf\x52\xa2\x59\xcb\xd2\xb0\x5b\xb2\x13\x95\x7e\x7d\x91\x96\xde\x66\x98\xc7\xe4\x6a\x2f\x42\x5a\x0a\xd3\x06\x9f\x42\x58\x2b\x27\x30\xed\x52\xd1\x52\xf9\x4d\xb2\x91\x1a\x88\x4f\x71\xf8\x05\xa9\xb9\xe8\xe3\x4a\xb7\x3e\x10\x11\xe1\xc4\x48\xab\x29\x58\x31\xc7\x6c\xab\xc7\x7f\xbf\x57\x3b\x24\xcf\x51\x72\x1c\xbf\xd8\x9e\xac\xf2\xe6\xb9\x73\xb5\x63\x49\xa5\x74\x61\x98\x42\x10\x75\xae\x20\x51\xd8\x75\xdd\x6f\xf0\x81\xf6\x54\x1a\x7b\x1f\x2b\x2f\x4d\x4a\x8e\xc3\x14\x3e\x01\x00\x00\xff\xff\x18\x75\x72\x51\xb5\x00\x00\x00")

func migrations3_snowballUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations3_snowballUpSql,
		"migrations/3_snowball.up.sql",
	)
}

func migrations3_snowballUpSql() (*asset, error) {
	bytes, err := migrations3_snowballUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/3_snowball.up.sql", size: 181, mode: os.FileMode(436), modTime: time.Unix(1571435094, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations4_stemmerstateDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\x29\xca\x2f\x50\x28\x49\x4c\xca\x49\x55\x28\x2e\x49\xcd\xcd\x4d\x2d\x2a\x2e\x49\x2c\x49\xb5\xe6\x02\x04\x00\x00\xff\xff\x40\x10\xb9\x1c\x19\x00\x00\x00")

func migrations4_stemmerstateDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations4_stemmerstateDownSql,
		"migrations/4_stemmerstate.down.sql",
	)
}

func migrations4_stemmerstateDownSql() (*asset, error) {
	bytes, err := migrations4_stemmerstateDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/4_stemmerstate.down.sql", size: 25, mode: os.FileMode(436), modTime: time.Unix(1571435094, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations4_stemmerstateUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xd0\x4b\x4e\xc4\x30\x10\x04\xd0\xbd\x4f\xd1\x4b\x90\x72\x83\x88\x15\xdc\x63\xd4\x71\x2a\xc6\xc2\x9f\xa8\xbb\x8c\xe6\xf8\x88\x11\x0c\xf3\x13\xeb\x2a\xd9\xf5\x3a\x1a\x94\x10\xea\x52\x20\x79\x93\xd6\x29\x38\x66\xa7\x8b\x13\xb5\xc2\x9c\xdf\x85\xa7\x20\x22\x52\xb4\xa5\xa1\x09\x2e\xc4\x91\xa7\x72\x1b\xa5\x4c\xa7\xd0\x50\xfb\x27\xde\xb2\x46\xcb\xcc\xd1\x65\xe9\xbd\x40\xdb\x4d\x8d\xfd\x03\xed\xf5\x5d\x4d\x23\x61\x0f\x5f\x72\xec\x6a\xca\xfe\x38\x1d\xfb\xaa\xc4\x2a\xcc\x15\x4e\xad\xbb\xac\xd8\x74\x14\x4a\x1c\x66\x68\x3c\x9c\x93\xf0\x3c\x87\xf0\x4b\xb4\x9c\x12\xec\x1f\xe4\x41\x87\xe8\x46\xd8\xcf\x17\xa1\x6f\x7f\xe2\xe9\xce\x37\xdd\x52\xa6\x8b\xe1\xa1\xb7\xeb\x03\x2e\x48\xb9\x5d\xec\xbf\x4e\x1d\x3c\xbb\x5e\xee\x1d\x73\x40\x5b\xe7\xf0\x15\x00\x00\xff\xff\xf5\xd9\x27\x17\xad\x01\x00\x00")

func migrations4_stemmerstateUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations4_stemmerstateUpSql,
		"migrations/4_stemmerstate.up.sql",
	)
}

func migrations4_stemmerstateUpSql() (*asset, error) {
	bytes, err := migrations4_stemmerstateUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/4_stemmerstate.up.sql", size: 429, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations5_spaceindexDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\x29\xca\x2f\x50\xc8\xcc\x4b\x49\xad\x50\x48\xc9\x4f\x2e\x8e\x2f\x2e\x48\x4c\x4e\x05\xf3\xad\xb9\x00\x01\x00\x00\xff\xff\xb4\x81\x4f\x9f\x1c\x00\x00\x00")

func migrations5_spaceindexDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations5_spaceindexDownSql,
		"migrations/5_spaceindex.down.sql",
	)
}

func migrations5_spaceindexDownSql() (*asset, error) {
	bytes, err := migrations5_spaceindexDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/5_spaceindex.down.sql", size: 28, mode: os.FileMode(436), modTime: time.Unix(1572301520, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations5_spaceindexUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\x2e\x4a\x4d\x2c\x49\x55\xc8\xcc\x4b\x49\xad\x50\xc8\x4c\x53\xc8\xcb\x2f\x51\x48\xad\xc8\x2c\x2e\x29\x56\x48\xc9\x4f\x2e\x8e\x2f\x2e\x48\x4c\x4e\x05\xcb\x72\xe5\xe7\x81\x85\x34\xc0\x42\x9e\x2e\x9a\xd6\x5c\x80\x00\x00\x00\xff\xff\xeb\xd3\x8d\x25\x3d\x00\x00\x00")

func migrations5_spaceindexUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations5_spaceindexUpSql,
		"migrations/5_spaceindex.up.sql",
	)
}

func migrations5_spaceindexUpSql() (*asset, error) {
	bytes, err := migrations5_spaceindexUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/5_spaceindex.up.sql", size: 61, mode: os.FileMode(436), modTime: time.Unix(1572301506, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations6_prefixindexDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\xcc\xc1\x0a\x83\x40\x0c\x04\xd0\x7b\xbe\x22\xb7\x28\x78\xed\x49\xfc\x96\xb2\xba\xb1\x84\x2e\x49\xd9\x64\x55\xfa\xf5\x65\x69\xe9\x6d\x86\x79\x4c\xae\xf6\xc2\x48\x6b\x61\xdc\xc3\x67\x80\xad\x72\x0a\xc6\x43\x6a\xb4\x54\x7e\x93\xec\xa8\x16\xc8\x97\x78\x78\x87\xd8\x5c\xf4\xd1\xd3\x6d\x00\x44\xc4\xb8\x62\xc2\xcd\x34\x58\x63\xa1\x6c\x9b\xd3\xbf\xdf\xab\x9d\x92\x17\x92\x4c\xd3\x17\xdb\x93\x55\xde\xbc\x90\xab\x9d\x6b\x2a\x85\x60\x9c\x01\x44\x9d\x6b\xa0\x68\x58\xbf\x1e\xf6\xf0\x11\x8f\x54\x1a\xfb\x40\x95\xd7\x26\x25\xd3\x38\xc3\x27\x00\x00\xff\xff\xd6\xf9\xe1\xa0\xb5\x00\x00\x00")

func migrations6_prefixindexDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations6_prefixindexDownSql,
		"migrations/6_prefixindex.down.sql",
	)
}

func migrations6_prefixindexDownSql() (*asset, error) {
	bytes, err := migrations6_prefixindexDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/6_prefixindex.down.sql", size: 181, mode: os.FileMode(436), modTime: time.Unix(1572894572, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations6_prefixindexUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\xcd\x4d\x6a\xc5\x30\x0c\x04\xe0\xbd\x4f\x31\x3b\x25\x90\x55\x7f\x56\x21\x67\x29\x4e\x2c\x17\x51\x23\x3d\x2c\xf9\x25\xf4\xf4\x25\xb4\x74\x37\xc3\x7c\x30\xa5\xdb\x03\x91\xf7\xc6\xa8\xe1\x6b\x4a\x47\xe7\x1c\x8c\xa7\xf4\x18\xb9\xfd\x4d\x52\xa1\x16\xe0\x4b\x3c\xfc\x86\x18\x2e\xfa\x79\xa7\xf7\x29\x01\x40\x5c\xb1\xe0\x30\x0d\xd6\xd8\xa8\xd8\xe1\xf4\xdf\x3f\xba\x9d\x52\x36\x92\x42\xcb\x2f\xb6\x2f\x56\xf9\xe6\x8d\x5c\xed\xdc\x73\x6b\xb4\xe0\xd1\xb9\xca\xb5\xd1\x0b\x5e\xf1\x46\x69\x5e\x53\x12\x75\xee\x01\xd1\xb0\xfb\x6a\xaa\xe1\x33\x9e\xb9\x0d\xf6\x89\x3a\xef\x43\x5a\xa1\x79\x4d\x3f\x01\x00\x00\xff\xff\xbc\x67\xa2\xb3\xc5\x00\x00\x00")

func migrations6_prefixindexUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations6_prefixindexUpSql,
		"migrations/6_prefixindex.up.sql",
	)
}

func migrations6_prefixindexUpSql() (*asset, error) {
	bytes, err := migrations6_prefixindexUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/6_prefixindex.up.sql", size: 197, mode: os.FileMode(436), modTime: time.Unix(1572896163, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations7_interesttimeDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x04\xc0\xd1\x09\x42\x31\x0c\x05\xd0\x7f\xa7\xb8\x3c\x78\x44\x41\x5c\xc0\x61\x24\x36\xb7\x20\xa4\x8d\xb4\x29\xfe\x74\x78\x8f\x7a\x72\x20\xf5\xed\x84\x45\x99\x18\xec\xda\x88\x12\xbe\x5a\xc7\xfa\x9a\x26\x0d\x19\x38\x8c\xce\xa4\xbd\x0e\xec\x8d\x99\xa3\xe6\xa7\xf1\x2a\xe7\x7c\x9c\x55\xee\xd2\xe3\x27\xb7\xe7\xe5\x1f\x00\x00\xff\xff\xa0\x00\x38\x5b\x51\x00\x00\x00")

func migrations7_interesttimeDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations7_interesttimeDownSql,
		"migrations/7_interesttime.down.sql",
	)
}

func migrations7_interesttimeDownSql() (*asset, error) {
	bytes, err := migrations7_interesttimeDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/7_interesttime.down.sql", size: 81, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations7_interesttimeUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x1c\xca\xc1\x0d\x84\x50\x08\x04\xd0\xfb\x56\x31\x25\xec\xdd\x1e\xec\x01\x65\x34\x26\xc8\x37\xfc\xa1\x7f\x13\xef\xcf\x42\x2c\xc8\xb6\x20\xae\x14\x8b\x53\x30\x77\xec\x23\xfa\x4e\xf4\xe3\x26\xfa\x6a\x39\xe6\x07\x4e\x16\x72\x08\xd9\x11\x70\x1e\xd6\x21\xfc\x97\xdf\x1b\x00\x00\xff\xff\x2f\x59\xb5\x3c\x49\x00\x00\x00")

func migrations7_interesttimeUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations7_interesttimeUpSql,
		"migrations/7_interesttime.up.sql",
	)
}

func migrations7_interesttimeUpSql() (*asset, error) {
	bytes, err := migrations7_interesttimeUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/7_interesttime.up.sql", size: 73, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations8_indexidDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\x29\xca\x2f\x50\x28\x49\x4c\xca\x49\x55\xc8\x4d\x2d\x49\xb4\xe6\x02\x04\x00\x00\xff\xff\x46\xdf\x11\xc4\x11\x00\x00\x00")

func migrations8_indexidDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations8_indexidDownSql,
		"migrations/8_indexid.down.sql",
	)
}

func migrations8_indexidDownSql() (*asset, error) {
	bytes, err := migrations8_indexidDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/8_indexid.down.sql", size: 17, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _migrations8_indexidUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\xcb\x31\x0a\xc3\x30\x10\x04\xc0\x5e\xaf\xd8\x32\x81\xfc\x20\x6d\x9a\xbc\x22\x5c\xa4\x35\x08\x4e\x67\x23\xaf\x40\xcf\x37\xd8\xe0\x7a\x98\xdc\x69\x22\x64\x7f\x27\x1a\x65\x78\x24\x00\xa8\x51\x38\xbf\x1f\x88\x53\x88\x55\x88\xe1\xfe\x3a\xe9\x2a\x05\xaa\x8d\xbb\xac\x6d\x28\x5c\x6c\xb8\x90\x47\xef\x0c\xfd\x6e\x49\xcf\x77\x3a\x02\x00\x00\xff\xff\xf5\x96\x99\x70\x62\x00\x00\x00")

func migrations8_indexidUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations8_indexidUpSql,
		"migrations/8_indexid.up.sql",
	)
}

func migrations8_indexidUpSql() (*asset, error) {
	bytes, err := migrations8_indexidUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/8_indexid.up.sql", size: 98, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _queriesSearch_1Sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x52\x41\x6e\x1b\x31\x0c\xbc\xeb\x15\x03\x5f\x22\x15\xaa\x61\x5f\x0d\xa4\x45\x81\x5c\xf2\x83\x5e\x15\x49\x1b\xab\x5e\x4b\x0b\x91\xae\x53\xc0\x87\xbe\xa6\x0f\xeb\x4b\x0a\x51\xbb\x8e\xd1\x78\x0f\x4b\x69\xc8\x11\x39\x24\xcf\x89\xf7\xea\xe8\xd8\xef\x23\xc1\x11\xb4\x02\x00\x8a\x63\xf4\x2c\xc7\xf6\xd5\x72\x4e\xc1\x5e\xaf\x43\xaa\xc4\xc2\xd1\x03\x93\x69\x34\x81\xde\x23\xaa\xcb\x87\x06\x57\x41\x86\x5a\x8e\xef\x64\x26\x39\x9f\xf7\xb1\xc6\x5b\x14\xf2\x22\x76\x62\xc4\x31\xa6\x63\x62\xec\xbc\x9b\x94\xb1\x8a\xd8\xf1\x87\x12\xe1\xcb\x29\xb3\xfe\x24\x45\xf8\xcc\x92\x0b\xb3\x1e\x65\x94\xba\x51\x42\x93\xf3\xd1\xa2\x4a\x61\x2e\x1f\xac\x10\x1c\x81\x0b\xbb\xd1\xe2\x47\x49\x39\x86\x75\x28\xfe\xf9\xa9\xc1\x8b\x64\x3a\xbd\x10\x57\xbd\xfa\xfb\xfb\xcf\xca\x62\x6b\xa1\x45\x2d\xbe\x60\x63\xcc\xe5\x52\xe3\x34\x3a\x1f\xf5\x6b\x64\x2e\x87\x98\xa9\x35\xc5\x22\x14\x4f\x6b\x7e\x63\x8b\xa3\x7b\xeb\x8c\xcf\x5b\x8b\x8d\xb1\xd8\xb6\xdf\xf7\x87\xcd\xb7\x07\x8b\x15\x56\xe6\x72\x91\xb7\x5b\x4e\xca\x69\x9a\x22\x2b\x91\x71\x77\x16\xb3\x8a\xde\x70\x54\x0b\xe9\xcb\xda\x67\x9e\x73\x4a\xfd\xf3\x39\x85\x8f\x03\x58\x9a\xb3\xdc\xc7\x38\xb0\x68\x17\x0a\x4a\x5e\xa8\x78\x5c\x62\xd7\xb2\x00\x57\x86\xaf\x85\xa8\x53\x24\xf9\xd5\xd1\xa1\x56\x20\xe1\x44\x29\xbf\x6a\xb9\x3c\x3f\x99\x3b\x23\x17\x17\x52\x86\xfe\x6a\xae\xa0\xcb\xa1\xa7\x77\x63\xfa\xd9\x63\x4b\x0d\xb1\xe2\xe5\x97\x0c\xce\xdf\x2e\x86\x98\x1e\x33\x0c\x14\x19\xbb\x6e\x95\x99\x67\xa9\xfe\xd3\xd6\x6b\x4a\xc1\xdc\x38\xda\xe6\x15\x31\x5d\x25\x1e\xa1\xe7\xdd\x4a\xa1\xaf\x93\x70\x7b\xce\xad\x51\xff\x02\x00\x00\xff\xff\x4f\xfe\x70\x6e\x32\x03\x00\x00")

func queriesSearch_1SqlBytes() ([]byte, error) {
	return bindataRead(
		_queriesSearch_1Sql,
		"queries/search_1.sql",
	)
}

func queriesSearch_1Sql() (*asset, error) {
	bytes, err := queriesSearch_1SqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "queries/search_1.sql", size: 818, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _queriesSearch_2Sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x64\x52\x41\x6e\x1b\x31\x0c\xbc\xeb\x15\x03\x5f\x22\x15\xea\xc2\xbe\xba\x48\x8b\x02\xb9\xe4\x07\xbd\x2a\x92\x36\x56\xbd\x96\x16\x22\x5d\xa7\x80\x0f\x7d\x4d\x1f\xd6\x97\x14\xa2\xd6\xd9\xb4\xf1\xc1\x94\x28\xce\x90\x33\xdc\x4b\xe2\x83\x3a\x39\xf6\x87\x48\x70\x04\xad\x00\x80\xe2\x14\x3d\xcb\xb1\xfd\x6a\xb9\xa4\x60\x5f\xaf\x63\xaa\xc4\x82\xd1\x23\x93\x69\x30\x49\xad\x15\xd5\xe5\x63\x4b\x57\xc9\x8c\xb5\x9c\x56\x30\x93\x9c\x2f\x87\x58\xe3\xdb\x2c\x84\x11\x7b\x09\xf2\x30\xa5\x53\x62\xec\xbd\x9b\x95\xb1\x8a\xd8\xf1\xbb\x11\xe1\xcb\x39\xb3\xfe\x20\x43\xf8\xcc\xd2\x0b\x8b\x1e\x65\xd4\x1b\x21\x34\x3b\x1f\x69\x90\x60\x11\x8a\xa7\x21\x14\xff\xf8\xd0\x90\x29\xd8\x1b\x68\xa8\x32\xb8\xcb\x47\x0b\x69\x39\x34\x5a\x47\xe0\xc2\x6e\xea\x12\xe9\xfc\x44\x5c\xf5\xe6\xcf\xaf\xdf\x1b\x8b\x9d\x85\x16\xfd\xf8\x8c\xad\x31\xd7\x6b\x8d\xf3\xe4\x7c\xd4\xcf\x91\xb9\x1c\x63\xa6\x66\xd3\xd2\x92\x5f\xb8\xb5\x7a\xe9\x88\x8f\x3b\x8b\xad\xb1\xd8\xb5\xbf\x6f\x77\xdb\xaf\x77\x16\x1b\x6c\xcc\xf5\x2a\xdc\xad\x2d\xe5\x34\xcf\x91\xd5\xab\x89\x37\x71\xed\xfc\xbd\xa4\x2c\xbc\x28\x3d\x0e\x29\xe0\x7e\x95\xd2\xf6\xd6\xad\x8c\x23\xf7\xea\xe6\x74\x91\xd0\x9f\x71\x0f\xbd\x78\x99\x42\xb7\x4f\x08\xbb\xf9\x3b\xf3\x1f\xbc\xbb\x88\x33\xa5\xfc\x0c\x2d\xb7\xc7\x87\x5e\xe4\x6b\x21\x5a\xaa\x9a\x71\x6a\x5d\xb1\x8c\xe6\xa6\xf4\xa3\x5f\x5d\x0e\x9d\x08\x29\x43\x7f\x31\xaa\xd4\x10\x2b\x9e\x7e\xfe\xb3\x04\x7f\xfb\x00\x7a\x28\xe3\x48\x91\xb1\xef\xf1\x93\xfa\x1b\x00\x00\xff\xff\xb2\x86\x74\xc9\xba\x02\x00\x00")

func queriesSearch_2SqlBytes() ([]byte, error) {
	return bindataRead(
		_queriesSearch_2Sql,
		"queries/search_2.sql",
	)
}

func queriesSearch_2Sql() (*asset, error) {
	bytes, err := queriesSearch_2SqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "queries/search_2.sql", size: 698, mode: os.FileMode(436), modTime: time.Unix(1573422571, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"migrations/1_init.down.sql":         migrations1_initDownSql,
	"migrations/1_init.up.sql":           migrations1_initUpSql,
	"migrations/2_alive.down.sql":        migrations2_aliveDownSql,
	"migrations/2_alive.up.sql":          migrations2_aliveUpSql,
	"migrations/3_snowball.down.sql":     migrations3_snowballDownSql,
	"migrations/3_snowball.up.sql":       migrations3_snowballUpSql,
	"migrations/4_stemmerstate.down.sql": migrations4_stemmerstateDownSql,
	"migrations/4_stemmerstate.up.sql":   migrations4_stemmerstateUpSql,
	"migrations/5_spaceindex.down.sql":   migrations5_spaceindexDownSql,
	"migrations/5_spaceindex.up.sql":     migrations5_spaceindexUpSql,
	"migrations/6_prefixindex.down.sql":  migrations6_prefixindexDownSql,
	"migrations/6_prefixindex.up.sql":    migrations6_prefixindexUpSql,
	"migrations/7_interesttime.down.sql": migrations7_interesttimeDownSql,
	"migrations/7_interesttime.up.sql":   migrations7_interesttimeUpSql,
	"migrations/8_indexid.down.sql":      migrations8_indexidDownSql,
	"migrations/8_indexid.up.sql":        migrations8_indexidUpSql,
	"queries/search_1.sql":               queriesSearch_1Sql,
	"queries/search_2.sql":               queriesSearch_2Sql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"migrations": &bintree{nil, map[string]*bintree{
		"1_init.down.sql":         &bintree{migrations1_initDownSql, map[string]*bintree{}},
		"1_init.up.sql":           &bintree{migrations1_initUpSql, map[string]*bintree{}},
		"2_alive.down.sql":        &bintree{migrations2_aliveDownSql, map[string]*bintree{}},
		"2_alive.up.sql":          &bintree{migrations2_aliveUpSql, map[string]*bintree{}},
		"3_snowball.down.sql":     &bintree{migrations3_snowballDownSql, map[string]*bintree{}},
		"3_snowball.up.sql":       &bintree{migrations3_snowballUpSql, map[string]*bintree{}},
		"4_stemmerstate.down.sql": &bintree{migrations4_stemmerstateDownSql, map[string]*bintree{}},
		"4_stemmerstate.up.sql":   &bintree{migrations4_stemmerstateUpSql, map[string]*bintree{}},
		"5_spaceindex.down.sql":   &bintree{migrations5_spaceindexDownSql, map[string]*bintree{}},
		"5_spaceindex.up.sql":     &bintree{migrations5_spaceindexUpSql, map[string]*bintree{}},
		"6_prefixindex.down.sql":  &bintree{migrations6_prefixindexDownSql, map[string]*bintree{}},
		"6_prefixindex.up.sql":    &bintree{migrations6_prefixindexUpSql, map[string]*bintree{}},
		"7_interesttime.down.sql": &bintree{migrations7_interesttimeDownSql, map[string]*bintree{}},
		"7_interesttime.up.sql":   &bintree{migrations7_interesttimeUpSql, map[string]*bintree{}},
		"8_indexid.down.sql":      &bintree{migrations8_indexidDownSql, map[string]*bintree{}},
		"8_indexid.up.sql":        &bintree{migrations8_indexidUpSql, map[string]*bintree{}},
	}},
	"queries": &bintree{nil, map[string]*bintree{
		"search_1.sql": &bintree{queriesSearch_1Sql, map[string]*bintree{}},
		"search_2.sql": &bintree{queriesSearch_2Sql, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
