// +build profile

package letarette

import (
	_ "net/http/pprof"

	"github.com/erkkah/letarette/pkg/logger"
)

func init() {
	logger.Info.Printf("Exposing pprof data at /debug/pprof")
}
