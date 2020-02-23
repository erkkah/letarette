package main

import (
	"fmt"
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
}
