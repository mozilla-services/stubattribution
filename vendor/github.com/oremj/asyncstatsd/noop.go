package asyncstatsd

import (
	"time"

	"github.com/alexcesaro/statsd"
)

type noopclient struct{}

// NewNoop returns a new noop client
func NewNoop() Client {
	return new(noopclient)
}

func (c *noopclient) Count(bucket string, n interface{}) {
}

func (c *noopclient) Gauge(bucket string, value interface{}) {
}

func (c *noopclient) Increment(bucket string) {
}

func (c *noopclient) Histogram(bucket string, value interface{}) {
}

func (c *noopclient) Timing(bucket string, value interface{}) {
}

func (c *noopclient) Clone(opts ...statsd.Option) Client {
	return c
}

type nooptiming struct{}

func (c *noopclient) NewTiming() Timing {
	return nooptiming{}
}

func (t nooptiming) Send(bucket string) {
}

func (t nooptiming) Duration() time.Duration {
	return 0
}
