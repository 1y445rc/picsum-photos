package queue

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

// Queue is a worker queue with a fixed amount of workers
type Queue struct {
	workers int
	queue   chan job
	handler func(context.Context, interface{}) (interface{}, error)
	ctx     context.Context
}

type job struct {
	data    interface{}
	result  chan jobResult
	context context.Context
}

type jobResult struct {
	result interface{}
	err    error
}

// New creates a new Queue with the specified amount of workers
func New(ctx context.Context, workers int, queueSize int, handler func(context.Context, interface{}) (interface{}, error)) *Queue {
	queue := &Queue{
		workers: workers,
		queue:   make(chan job, queueSize),
		handler: handler,
		ctx:     ctx,
	}

	return queue
}

// Run starts the queue and blocks until it's shut down
func (q *Queue) Run() {
	for i := 0; i < q.workers; i++ {
		go q.worker()
	}

	<-q.ctx.Done()
	close(q.queue)
}

func (q *Queue) worker() {
	// Lock the thread to ensure that we get our own thread, and that tasks aren't moved between threads
	// We won't unlock since it's uncertain how libvips would react
	runtime.LockOSThread()

	for {
		select {
		case job, open := <-q.queue:
			if !open {
				return
			}

			select {
			// End early if the job context was cancelled
			case <-job.context.Done():
				job.result <- jobResult{
					result: nil,
					err:    job.context.Err(),
				}
			// Otherwise run the job
			default:
				result, err := q.handler(job.context, job.data)
				job.result <- jobResult{
					result: result,
					err:    err,
				}
			}

		case <-q.ctx.Done():
			return
		}
	}
}

// Process adds a job to the queue, waits for it to process, and returns the result
func (q *Queue) Process(ctx context.Context, data interface{}) (interface{}, error) {
	if q.ctx.Err() != nil {
		return nil, fmt.Errorf("queue has been shutdown")
	}

	resultChan := make(chan jobResult)
	defer close(resultChan)

	select {
	// Attempt to queue the job
	case q.queue <- job{
		data:    data,
		result:  resultChan,
		context: ctx,
	}:
		result := <-resultChan

		if result.err != nil {
			return nil, result.err
		}

		return result.result, nil
	// If the queue is full of jobs waiting to be processed, just drop the job and return an error
	default:
		return nil, ErrQueueFull
	}
}

var (
	ErrQueueFull = errors.New("queue full")
)
