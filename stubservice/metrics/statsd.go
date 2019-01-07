package metrics

import (
	"os"

	"github.com/oremj/gostatsd/statsd"
	"github.com/sirupsen/logrus"
)

// Statsd exposes a global statsd client
var Statsd *statsd.Client

func mustStatsd(opts ...statsd.Option) *statsd.Client {
	c, err := statsd.New(opts...)
	if err != nil {
		logrus.WithError(err).Fatal("Could not initiate statsd")
	}
	return c
}

func init() {
	statsdPrefix := os.Getenv("STATSD_PREFIX")
	statsdAddr := os.Getenv("STATSD_ADDR")
	if statsdAddr == "" {
		statsdAddr = "127.0.0.1:8125"
	}
	if statsdPrefix == "" {
		statsdPrefix = "stubattribution"
	}
	Statsd = mustStatsd(
		statsd.Prefix(statsdPrefix),
		statsd.Address(statsdAddr),
		statsd.TagsFormat(statsd.Datadog),
	)
}
