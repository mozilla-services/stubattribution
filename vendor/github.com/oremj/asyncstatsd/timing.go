package asyncstatsd

import "time"

type Timing interface {
	Duration() time.Duration
	Send(bucket string)
}
