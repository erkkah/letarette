package letarette

import (
	"expvar"
	"fmt"
	"net/http"

	"github.com/zserge/metric"
)

var metrics = struct {
	docRequests metric.Metric
}{
	docRequests: metric.NewCounter("1m5s", "5m10s", "15m10s"),
}

// ExposeMetrics is a test for exposing metrics
func ExposeMetrics(port uint16) {
	expvar.Publish("doc:requests", metrics.docRequests)

	http.Handle("/debug/metrics", metric.Handler(metric.Exposed))
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}()
}
