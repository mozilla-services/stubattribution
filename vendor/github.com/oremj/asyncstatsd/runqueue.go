package asyncstatsd

import "github.com/sirupsen/logrus"

type RunQueue struct {
	ch chan func()
}

// New returns a new runqueue
func NewRunQueue(size int) *RunQueue {
	q := &RunQueue{
		ch: make(chan func(), size),
	}
	go q.loop()
	return q
}

// Queue queues a function to be run
func (r *RunQueue) Queue(f func()) {
	select {
	case r.ch <- f:
	default:
		logrus.Error("runqueue is full, dropping function")
	}
}

func (r *RunQueue) loop() {
	for f := range r.ch {
		f()
	}
}

// Close closes the run queue channel
func (r *RunQueue) Close() {
	close(r.ch)
}
