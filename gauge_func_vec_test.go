package gaugefuncvec_test

import (
	"strings"
	"testing"

	"github.com/colega/gaugefuncvec"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestGaugeFuncVec_MustRegister(t *testing.T) {
	t.Run("with const labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace:   "gaugefuncvec",
				Subsystem:   "test",
				Name:        "with_const_labels",
				Help:        "A vector of gauge funcs with const labels",
				ConstLabels: prometheus.Labels{"const": "label"},
			},
			[]string{"number"},
		)
		g.MustRegister(
			prometheus.Labels{"number": "one"},
			func() float64 { return 1 },
		)
		g.MustRegister(
			prometheus.Labels{"number": "two"},
			func() float64 { return 2 },
		)

		reg := prometheus.NewRegistry()
		reg.MustRegister(g)

		gatherAndCompare(
			t,
			reg,
			`
			# HELP gaugefuncvec_test_with_const_labels A vector of gauge funcs with const labels
			# TYPE gaugefuncvec_test_with_const_labels gauge
			gaugefuncvec_test_with_const_labels{const="label",number="one"} 1
			gaugefuncvec_test_with_const_labels{const="label",number="two"} 2
			`,
			"gaugefuncvec_test_with_const_labels",
		)
	})

	t.Run("without const labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace: "gaugefuncvec",
				Subsystem: "test",
				Name:      "with_const_labels",
				Help:      "A vector of gauge funcs",
			},
			[]string{"number"},
		)
		g.MustRegister(
			prometheus.Labels{"number": "ten"},
			func() float64 { return 10 },
		)
		g.MustRegister(
			prometheus.Labels{"number": "twenty"},
			func() float64 { return 20 },
		)

		reg := prometheus.NewRegistry()
		reg.MustRegister(g)

		gatherAndCompare(
			t,
			reg,
			`
			# HELP gaugefuncvec_test_with_const_labels A vector of gauge funcs
			# TYPE gaugefuncvec_test_with_const_labels gauge
			gaugefuncvec_test_with_const_labels{number="ten"} 10
			gaugefuncvec_test_with_const_labels{number="twenty"} 20
			`,
			"gaugefuncvec_test_with_const_labels",
		)
	})

	t.Run("without any labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace: "gaugefuncvec",
				Subsystem: "test",
				Name:      "without_labels",
				Help:      "A single gauge without labels, just like prometheus.GaugeFunc()",
			},
			nil,
		)
		g.MustRegister(
			prometheus.Labels{},
			func() float64 { return 288 },
		)

		reg := prometheus.NewRegistry()
		reg.MustRegister(g)

		gatherAndCompare(
			t,
			reg,
			`
			# HELP gaugefuncvec_test_without_labels A single gauge without labels, just like prometheus.GaugeFunc()
			# TYPE gaugefuncvec_test_without_labels gauge
			gaugefuncvec_test_without_labels 288
			`,
			"gaugefuncvec_test_without_labels",
		)
	})

	t.Run("metric function is really called every time", func(t *testing.T) {
		v := 100.0

		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace: "gaugefuncvec",
				Subsystem: "test",
				Name:      "with_changing_value",
				Help:      "A vector of gauge funcs",
			},
			[]string{"number"},
		)
		g.MustRegister(
			prometheus.Labels{"number": "changing"},
			func() float64 {
				v++
				return v
			},
		)

		reg := prometheus.NewRegistry()
		reg.MustRegister(g)

		gatherAndCompare(
			t,
			reg,
			`
			# HELP gaugefuncvec_test_with_changing_value A vector of gauge funcs
			# TYPE gaugefuncvec_test_with_changing_value gauge
			gaugefuncvec_test_with_changing_value{number="changing"} 101
			`,
			"gaugefuncvec_test_with_changing_value",
		)

		gatherAndCompare(
			t,
			reg,
			`
			# HELP gaugefuncvec_test_with_changing_value A vector of gauge funcs
			# TYPE gaugefuncvec_test_with_changing_value gauge
			gaugefuncvec_test_with_changing_value{number="changing"} 102
			`,
			"gaugefuncvec_test_with_changing_value",
		)
	})

	t.Run("panics with wrong labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace:   "gaugefuncvec",
				Subsystem:   "test",
				Name:        "with_same_labels",
				Help:        "A vector of gauge funcs with same labels",
				ConstLabels: prometheus.Labels{"label": "any"},
			},
			[]string{"expected"},
		)

		var panicked error
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					var ok bool
					panicked, ok = recovered.(error)
					if !ok {
						t.Errorf("Recovered panic is not an error, it is %v", recovered)
					}
				}
			}()
			g.MustRegister(
				prometheus.Labels{"unexpected": "label"},
				func() float64 { return 288 },
			)
		}()

		assertError(t, `labels don't include expected label expected, expected [expected], got [unexpected]`, panicked)
	})
}

func TestGaugeFuncVec_Register(t *testing.T) {
	t.Run("fails registering same labels twice", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace: "gaugefuncvec",
				Subsystem: "test",
				Name:      "with_same_labels",
				Help:      "A vector of gauge funcs with same labels",
			},
			[]string{"label"},
		)
		g.MustRegister(
			prometheus.Labels{"label": "same"},
			func() float64 { return 10 },
		)

		err := g.Register(
			prometheus.Labels{"label": "same"},
			func() float64 { return 20 },
		)

		assertError(t, `can't register again label values {label="same"}`, err)
	})

	t.Run("fails overriding const labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace:   "gaugefuncvec",
				Subsystem:   "test",
				Name:        "with_same_labels",
				Help:        "A vector of gauge funcs with same labels",
				ConstLabels: prometheus.Labels{"label": "any"},
			},
			[]string{"label"},
		)

		err := g.Register(
			prometheus.Labels{"label": "again"},
			func() float64 { return 20 },
		)

		assertError(t, `can't override const label label, const labels are {label="any"}`, err)
	})

	t.Run("fails with more labels than expected", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace:   "gaugefuncvec",
				Subsystem:   "test",
				Name:        "with_same_labels",
				Help:        "A vector of gauge funcs with same labels",
				ConstLabels: prometheus.Labels{"label": "any"},
			},
			[]string{"expected"},
		)

		err := g.Register(
			prometheus.Labels{"expected": "label", "unexpected": "label"},
			func() float64 { return 288 },
		)

		assertError(t, `unexpected number of labels, expected [expected], got [expected unexpected]`, err)
	})

	t.Run("fails with wrong labels", func(t *testing.T) {
		g := gaugefuncvec.New(
			prometheus.GaugeOpts{
				Namespace:   "gaugefuncvec",
				Subsystem:   "test",
				Name:        "with_same_labels",
				Help:        "A vector of gauge funcs with same labels",
				ConstLabels: prometheus.Labels{"label": "any"},
			},
			[]string{"expected"},
		)

		err := g.Register(
			prometheus.Labels{"unexpected": "label"},
			func() float64 { return 288 },
		)

		assertError(t, `labels don't include expected label expected, expected [expected], got [unexpected]`, err)
	})
}

func TestNew(t *testing.T) {
	t.Run("panics with labels overriding const labels", func(t *testing.T) {
		var panicked error
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					var ok bool
					panicked, ok = recovered.(error)
					if !ok {
						t.Errorf("Recovered panic is not an error, it is %v", recovered)
					}
				}
			}()
			_ = gaugefuncvec.New(
				prometheus.GaugeOpts{
					Namespace:   "gaugefuncvec",
					Subsystem:   "test",
					Name:        "with_const_labels",
					Help:        "A vector of gauge funcs with const labels",
					ConstLabels: prometheus.Labels{"some_label": "label"},
				},
				[]string{"some_label"},
			)
		}()

		assertError(t, `variableLabelNames shuold not include any of ConstLabels names, got variableLabelNames [some_label] including some_label defined in ConstLabels [some_label]`, panicked)
	})
}

func assertError(t *testing.T, expected string, err error) {
	t.Helper()
	if err == nil || err.Error() != expected {
		t.Errorf("Should fail with\n'%s'\ngot\n'%s'",
			expected,
			err,
		)
	}
}

func gatherAndCompare(t *testing.T, gatherer prometheus.Gatherer, expected string, metricNames ...string) {
	t.Helper()
	err := testutil.GatherAndCompare(
		gatherer,
		strings.NewReader(expected),
		metricNames...,
	)
	if err != nil {
		t.Error(err)
	}
}
