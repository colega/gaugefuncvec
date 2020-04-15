# Gauge Function Vector for Prometheus (golang)

[![Build Status](https://travis-ci.org/colega/gaugefuncvec.svg?branch=master)](https://travis-ci.org/colega/gaugefuncvec)
[![Coverage Status](https://coveralls.io/repos/github/colega/gaugefuncvec/badge.svg?branch=master)](https://coveralls.io/github/colega/gaugefuncvec?branch=master)
[![GoDoc](https://godoc.org/github.com/colega/gaugefuncvec?status.svg)](https://godoc.org/github.com/colega/gaugefuncvec)

## Summary

Like `prometheus.GaugeFunc` but with labels and one function per label.

## Why?

Let's say you want to observer your mysql driver stats, setting up a `prometheus.GaugeFunc` gathering your `db.Stats()` works great, however if you have several connections the idiomatic way is to label them, but you can't with `prometheus.GaugeFunc`

## How?

It works just as you'd expect to. `gaugevecfunc.New()` accepts the standard `prometheus.GaugeOpts` and respects them, additionally it accepts a slice of strings defining the labels your functions will have. It returns a `*prometheus.GaugeVecFunc` that is a `prometheus.Collector`, i.e., ready to be registered in the `prometheus.Registerer`.

The `GaugeVecFunc` provides two methods: `Register(labels []string, function func() float64) error` and `MustRegister(labels []string, function func())`. `MustRegister` invokes the `Register` and panics if an error was returned. 

`Register` will validate the labels, which should not collide with const labels and should respect the ones defined when `New()` was called.

## Example

See [`example_test`](./example_test.go) for a runnable golang example.

```go
	g := gaugefuncvec.New(
		prometheus.GaugeOpts{
			Namespace: "database",
			Name:      "connections",
			Help:      "Number of connections per database connection",
		},
		[]string{"connection_id"},
	)
	g.MustRegister(
		prometheus.Labels{"connection_id": "master"},
		func() float64 { return float64(db1.stats().conns) },
	)
	g.MustRegister(
		prometheus.Labels{"connection_id": "slave"},
		func() float64 { return float64(db2.stats().conns) },
	)
```

## Decisions

`New()` may not respect the order of the given variable labels.
Since `Register` isn't expected to be called frequently, it doesn't make sense to trade off readability of a `prometheus.Labels` for a performance of order-based labels.

There's no method to unregister. It's easy to implement, but there was no need when I wrote the library.

Different gather funcs could have been provided as constructor params to `New()` (as variadic options in a fancy way), however my usecase (instantiating db connections in separate places, when `GaugeFuncVec` is already instantiated and registered required a `Register` method, and I considered unnecesary to implement two ways of doing the same.
