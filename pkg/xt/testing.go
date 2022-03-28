// Copyright 2022 Erik Agsjö
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

package xt

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// XT is a testing.T - extension, adding a tiny bit
// of convenience, making tests more fun to write.
type XT struct {
	*testing.T
}

// X wraps a *testing.T and extends its functionality
func X(t *testing.T) XT {
	return XT{t}
}

// Assert verifies that a condition is true
func (x XT) Assert(cond bool) {
	x.Assertf(cond, "Assert failed")
}

// Assertf verifies that a condition is true
func (x XT) Assertf(cond bool, format string, msg ...interface{}) {
	if !cond {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// True verifies that a condition is true
func (x XT) True(cond bool) {
	x.Truef(cond, "Expected true")
}

// Truef verifies that a condition is true
func (x XT) Truef(cond bool, format string, msg ...interface{}) {
	if !cond {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// False verifies that a condition is false
func (x XT) False(cond bool) {
	x.Falsef(cond, "Expected false")
}

// Falsef verifies that a condition is false
func (x XT) Falsef(cond bool, format string, msg ...interface{}) {
	if cond {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// Equal verifies that the two arguments are equal
func (x XT) Equal(a interface{}, b interface{}) {
	x.Equalf(a, b, "%v should equal %v", a, b)
}

// Equalf verifies that the two arguments are equal
func (x XT) Equalf(a interface{}, b interface{}, format string, msg ...interface{}) {
	if a != b {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// NotEqual verifies that the two arguments are not equal
func (x XT) NotEqual(a interface{}, b interface{}, msg ...interface{}) {
	if a == b {
		x.Logf("%v should not equal %v", a, b)
		x.Log(msg...)
		x.Fail()
	}
}

// NotEqualf verifies that the two arguments are not equal
func (x XT) NotEqualf(a interface{}, b interface{}, format string, msg ...interface{}) {
	if a == b {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// Nil verifies that the argument is nil
func (x XT) Nil(a interface{}) {
	x.Nilf(a, "Expected nil, got %q", a)
}

// Nilf verifies that the argument is nil
func (x XT) Nilf(a interface{}, fmt string, msg ...interface{}) {
	if a != nil {
		x.Logf(fmt, msg...)
		x.Fail()
	}
}

// NotNil verifies that the argument is not nil
func (x XT) NotNil(a interface{}) {
	x.NotNilf(a, "Expected non-nil")
}

// NotNilf verifies that the argument is nil
func (x XT) NotNilf(a interface{}, fmt string, msg ...interface{}) {
	if a == nil {
		x.Logf(fmt, msg...)
		x.Fail()
	}
}

// Contains verifies that the argument stringed contains a given substring
func (x XT) Contains(a interface{}, needle string) {
	x.Containsf(a, needle, "Expected %q to contain %q", a, needle)
}

// Containsf verifies that the argument stringed contains a given substring
func (x XT) Containsf(a interface{}, needle string, format string, msg ...interface{}) {
	if !strings.Contains(fmt.Sprintf("%v", a), needle) {
		x.Logf(format, msg...)
		x.Fail()
	}
}

// DeepEqual verified that two arguments are deeply equal
func (x XT) DeepEqual(a interface{}, b interface{}) {
	x.DeepEqualf(a, b, "Expected %q to deeply equal %q", a, b)
}

// DeepEqualf verified that two arguments are deeply equal
func (x XT) DeepEqualf(a interface{}, b interface{}, format string, msg ...interface{}) {
	x.Assertf(reflect.DeepEqual(a, b), format, msg...)
}
