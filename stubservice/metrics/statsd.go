package metrics

import "github.com/oremj/asyncstatsd"

// Statsd exposes a global statsd client
var Statsd = asyncstatsd.NewNoop()
