package asyncstatsd

import "github.com/alexcesaro/statsd"

type Client interface {
	Count(bucket string, n interface{})
	Gauge(bucket string, value interface{})
	Increment(bucket string)
	Histogram(bucket string, value interface{})
	Timing(bucket string, value interface{})

	Clone(opts ...statsd.Option) Client
	NewTiming() Timing
}
