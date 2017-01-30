package asyncstatsd

import (
	"time"

	"github.com/alexcesaro/statsd"
)

type statsdclient struct {
	statsd *statsd.Client

	queue *RunQueue
}

// New returns a new statsdclient
func New(c *statsd.Client, queueSize int) Client {
	return &statsdclient{
		statsd: c,
		queue:  NewRunQueue(queueSize),
	}
}

func (c *statsdclient) Count(bucket string, n interface{}) {
	c.queue.Queue(func() {
		c.statsd.Count(bucket, n)
	})
}

func (c *statsdclient) Gauge(bucket string, value interface{}) {
	c.queue.Queue(func() {
		c.statsd.Gauge(bucket, value)
	})
}

func (c *statsdclient) Increment(bucket string) {
	c.queue.Queue(func() {
		c.statsd.Increment(bucket)
	})
}

func (c *statsdclient) Histogram(bucket string, value interface{}) {
	c.queue.Queue(func() {
		c.statsd.Histogram(bucket, value)
	})
}

func (c *statsdclient) Timing(bucket string, value interface{}) {
	c.queue.Queue(func() {
		c.statsd.Timing(bucket, value)
	})
}

func (c *statsdclient) Clone(opts ...statsd.Option) Client {
	c.statsd.Clone()
	return &statsdclient{
		statsd: c.statsd.Clone(opts...),
		queue:  c.queue,
	}
}

type statsdtiming struct {
	timing statsd.Timing
	c      Client
}

// Newstatsdtiming returns a new wrapped timing struct
func (c *statsdclient) NewTiming() Timing {
	return statsdtiming{
		timing: c.statsd.NewTiming(),
		c:      c,
	}
}

func (t statsdtiming) Send(bucket string) {
	t.c.Timing(bucket, int(t.Duration()/time.Millisecond))
}

func (t statsdtiming) Duration() time.Duration {
	return t.timing.Duration()
}
