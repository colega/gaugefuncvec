package gaugefuncvec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// GaugeFuncVec is a prometheus.Collector that allows registering GaugeFunc's for different labels
type GaugeFuncVec struct {
	desc *prometheus.Desc

	mtx     sync.RWMutex
	metrics map[string]*gaugeFunc

	constLabelPairs []*dto.LabelPair
	labelNames      []string
}

var (
	_ prometheus.Collector = &GaugeFuncVec{}
)

// New returns a GaugeFuncVec
// variableLabelNames should not include the ConstLabels defined in GaugeOpts, it will panic if they do
func New(opts prometheus.GaugeOpts, variableLabelNames []string) *GaugeFuncVec {
	if opts.ConstLabels != nil {
		for _, name := range variableLabelNames {
			if _, includes := opts.ConstLabels[name]; includes {
				panic(fmt.Errorf(
					"variableLabelNames shuold not include any of ConstLabels names, got variableLabelNames %v including %s defined in ConstLabels %s",
					variableLabelNames, name, labelNames(opts.ConstLabels),
				))
			}
		}
	}
	return &GaugeFuncVec{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
			opts.Help,
			variableLabelNames,
			opts.ConstLabels,
		),

		metrics: make(map[string]*gaugeFunc),

		constLabelPairs: labelPairs(opts.ConstLabels),
		labelNames:      variableLabelNames,
	}
}

// Describe implements prometheus.Collector
func (g *GaugeFuncVec) Describe(desc chan<- *prometheus.Desc) {
	desc <- g.desc
}

// Collect implements prometheus.Collector
func (g *GaugeFuncVec) Collect(metrics chan<- prometheus.Metric) {
	g.mtx.RLock()
	defer g.mtx.RUnlock()

	for _, metric := range g.metrics {
		metrics <- metric
	}
}

// MustRegister calls Register and panics if it returns an error
func (g *GaugeFuncVec) MustRegister(labels prometheus.Labels, function func() float64) {
	if err := g.Register(labels, function); err != nil {
		panic(err)
	}
}

// Register will register a function for the given set of labels
// Labels should respect the variable labels provided in New() and should not collide with const labels
func (g *GaugeFuncVec) Register(labels prometheus.Labels, function func() float64) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	funcLabelPairs := labelPairs(labels)
	pairs := make([]*dto.LabelPair, 0, len(g.constLabelPairs)+len(labels))
	pairs = append(pairs, g.constLabelPairs...)
	pairs = append(pairs, funcLabelPairs...)
	sort.Sort(labelPairsByName(pairs))

	gf := &gaugeFunc{
		labelPairs: pairs,
		desc:       g.desc,
		function:   function,
	}

	key := labelPairsToKey(gf.labelPairs)

	if len(labels) != len(g.labelNames) {
		return fmt.Errorf("unexpected number of labels, expected %v, got %v", g.labelNames, labelNames(labels))
	}

	for _, name := range g.labelNames {
		if _, defined := labels[name]; !defined {
			return fmt.Errorf("labels don't include expected label %s, expected %v, got %v", name, g.labelNames, labelNames(labels))
		}
	}

	for _, l := range g.constLabelPairs {
		if _, overrides := labels[l.GetName()]; overrides {
			return fmt.Errorf("can't override const label %s, const labels are %s", l.GetName(), labelPairsToKey(g.constLabelPairs))
		}
	}

	if _, exists := g.metrics[key]; exists {
		return fmt.Errorf("can't register again label values %s", key)
	}

	g.metrics[key] = gf
	return nil
}

type gaugeFunc struct {
	labelPairs []*dto.LabelPair
	desc       *prometheus.Desc
	function   func() float64
}

var _ prometheus.Metric = &gaugeFunc{}

func (gf *gaugeFunc) Desc() *prometheus.Desc {
	return gf.desc
}

func (gf *gaugeFunc) Write(m *dto.Metric) error {
	m.Label = gf.labelPairs
	v := gf.function()
	m.Gauge = &dto.Gauge{Value: proto.Float64(v)}
	return nil
}

func labelPairs(labels prometheus.Labels) []*dto.LabelPair {
	pairs := make([]*dto.LabelPair, 0, len(labels))
	for n, v := range labels {
		pairs = append(pairs, &dto.LabelPair{
			Name:  proto.String(n),
			Value: proto.String(v),
		})
	}
	return pairs
}

func labelNames(labels prometheus.Labels) []string {
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func labelPairsToKey(pairs []*dto.LabelPair) string {
	stringPairs := make([]string, 0, len(pairs))
	for _, p := range pairs {
		stringPairs = append(
			stringPairs,
			fmt.Sprintf("%s=%s", p.GetName(), strconv.Quote(p.GetValue())),
		)
	}
	return "{" + strings.Join(stringPairs, ",") + "}"
}

type labelPairsByName []*dto.LabelPair

func (s labelPairsByName) Len() int { return len(s) }

func (s labelPairsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s labelPairsByName) Less(i, j int) bool { return s[i].GetName() < s[j].GetName() }
