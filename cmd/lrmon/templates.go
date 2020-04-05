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
	"fmt"
	"math"
	"strconv"
	"time"
)

func anyToFloat(any interface{}) float64 {
	float, _ := strconv.ParseFloat(fmt.Sprintf("%v", any), 64)
	return float
}

var templateFunctions = map[string]interface{}{
	"add": func(increment, base interface{}) float64 {
		baseFloat := anyToFloat(base)
		incrementFloat := anyToFloat(increment)
		return baseFloat + incrementFloat
	},
	"list": func(arg ...interface{}) []interface{} {
		return arg
	},
	"time": func(format string, t time.Time) string {
		formatString := "2006-01-02 15:04:05 -07:00"
		switch format {
		case "time":
			formatString = "15:04:05"
		case "date":
			formatString = "2006-01-02"
		case "kitchen":
			formatString = time.Kitchen
		case "iso":
			fallthrough
		default:
		}
		return t.Format(formatString)
	},
	"SI": func(input interface{}) string {
		number := anyToFloat(input)
		siPrefixes := map[int]string{
			-15: "f",
			-12: "p",
			-9:  "n",
			-6:  "µ",
			-3:  "m",
			0:   "",
			3:   "k",
			6:   "M",
			9:   "G",
			12:  "T",
			15:  "P",
		}
		exp := math.Log10(number)
		exp = math.Floor(exp / 3)
		if exp > 5 {
			exp = 5
		}
		if exp < -5 {
			exp = -5
		}

		quantizedExponent := int(exp) * 3
		divider := math.Pow10(quantizedExponent)
		prefix := siPrefixes[quantizedExponent]

		return fmt.Sprintf("%.2f%s", number/divider, prefix)
	},
}
