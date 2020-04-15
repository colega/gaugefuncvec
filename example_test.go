package gaugefuncvec_test

import (
	"os"

	"github.com/colega/gaugefuncvec"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func Example() {
	g := gaugefuncvec.New(
		prometheus.GaugeOpts{
			Namespace: "database",
			Name:      "connections",
			Help:      "Number of connections per database connection",
		},
		[]string{"connection_id"},
	)

	db1 := &db{conns: 42}
	g.MustRegister(
		prometheus.Labels{"connection_id": "master"},
		func() float64 { return float64(db1.stats().conns) },
	)
	db2 := &db{conns: 288}
	g.MustRegister(
		prometheus.Labels{"connection_id": "slave"},
		func() float64 { return float64(db2.stats().conns) },
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(g)

	gatherAndPrintMetrics(registry)

	// Output:
	// # HELP database_connections Number of connections per database connection
	// # TYPE database_connections gauge
	// database_connections{connection_id="master"} 42
	// database_connections{connection_id="slave"} 288
}

func gatherAndPrintMetrics(gatherer prometheus.Gatherer) {
	metrics, _ := gatherer.Gather()
	enc := expfmt.NewEncoder(os.Stdout, expfmt.FmtText)
	for _, mf := range metrics {
		_ = enc.Encode(mf)
	}
}

type db struct{ conns int }

func (d db) stats() struct{ conns int } { return struct{ conns int }{d.conns} }
