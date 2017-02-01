package metrics

import "github.com/oremj/asyncstatsd"

var Statsd = asyncstatsd.NewNoop()
