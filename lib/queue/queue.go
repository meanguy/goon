package queue

import (
	"net"
	"time"
)

type (
	options struct {
		WorkerCount int
		Timeout     time.Duration
	}

	Payload net.Conn

	Queue struct {
		in   chan Payload
		outs []chan Payload
		next int
		conf options
	}
)

func NewQueue(
	workerCount int,
	bufferSize int,
	timeout time.Duration,
) *Queue {
	conf := options{
		WorkerCount: workerCount,
		Timeout:     timeout,
	}

	return &Queue{
		in:   make(chan Payload, 1),
		outs: make([]chan Payload, 0, conf.WorkerCount),
		next: 0,
		conf: conf,
	}
}

func (q *Queue) Close() {
	var c chan Payload

	for len(q.outs) > 0 {
		length := len(q.outs)
		c, q.outs = q.outs[length-1], q.outs[:length-1]
		close(c)
	}
}

// In returns a new InPipe for writing payloads to.
func (q *Queue) In() chan<- Payload {
	return q.in
}

func (q *Queue) Open() {
	go func() {
		for payload := range q.in {
			out := q.outs[q.next]
			q.next = (q.next + 1) % len(q.outs)

			out <- payload
		}
	}()
}

// Out returns a new OutPipe for reading payloads from.
func (q *Queue) Out() <-chan Payload {
	c := make(chan Payload, 1)
	q.outs = append(q.outs, c)

	return c
}
