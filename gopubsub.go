package gopubsub

import (
	"context"
	"errors"
	"sync"
)

const (
	defaultPublishChannelBufferSize   = 100
	defaultSubscribeChannelBufferSize = 100
)

var (
	ErrAlreadyStarted = errors.New("this topic has been already started")
	ErrAlreadyClosed  = errors.New("this topic has been already closed")
)

// Topic is a publish-subscribe topic.
type Topic[T any] struct {
	publishCh chan T
	closedCh  chan struct{}

	publishWg sync.WaitGroup

	subscriptionsMu struct {
		sync.RWMutex
		subscriptions []*subscription[T]
	}

	startedMu struct {
		sync.RWMutex
		started bool
	}

	closedMu struct {
		sync.RWMutex
		closed bool
	}
}

// NewTopic creates a new topic.
func NewTopic[T any]() *Topic[T] {
	t := &Topic[T]{
		publishCh: make(chan T, defaultPublishChannelBufferSize), // TODO: The buffer size should be configurable.
		closedCh:  make(chan struct{}),
	}

	return t
}

// Start starts the topic and blocks until the context is canceled.
// When the passed context is canceled, Start waits for all published messages to be processed by all subscribers.
// Once Start is called, then Start returns ErrAlreadyStarted.
func (t *Topic[T]) Start(ctx context.Context) error {
	t.startedMu.Lock()
	defer t.startedMu.Unlock()
	if t.startedMu.started {
		return ErrAlreadyStarted
	}
	t.startedMu.started = true

	go func() {
		defer close(t.closedCh)

		for {
			message, ok := <-t.publishCh
			if !ok {
				return
			}

			go func(message T) {
				defer t.publishWg.Done()

				var wg sync.WaitGroup

				t.subscriptionsMu.RLock()
				for _, s := range t.subscriptionsMu.subscriptions {
					wg.Add(1)
					go func(s *subscription[T]) {
						defer wg.Done()
						s.messageCh <- message
					}(s)
				}
				t.subscriptionsMu.RUnlock()

				wg.Wait()
			}(message)
		}
	}()

	<-ctx.Done()
	t.stop()

	return nil
}

// Publish publishes a message to the topic. This method is non-blocking and concurrent-safe.
// Publish returns ErrAlreadyClosed if the topic has been already closed.
func (t *Topic[T]) Publish(message T) error {
	t.closedMu.RLock()
	defer t.closedMu.RUnlock() // In order to avoid the race condition between Topic.Publish and Topic.stop, defer is necessary.
	if t.closedMu.closed {
		return ErrAlreadyClosed
	}

	t.publishWg.Add(1)
	t.publishCh <- message

	return nil
}

// Subscribe registers the passed function as a subscriber to the topic. This method is non-blocking and concurrent-safe.
// The function passed to Subscribe is called when a message is published to the topic.
// Subscribe returns ErrAlreadyClosed if the topic has been already closed.
func (t *Topic[T]) Subscribe(subscriber func(message T)) error {
	t.closedMu.RLock()
	if t.closedMu.closed {
		return ErrAlreadyClosed
	}
	t.closedMu.RUnlock()

	s := &subscription[T]{
		messageCh:    make(chan T, defaultSubscribeChannelBufferSize), // TODO: The buffer size should be configurable.
		terminatedCh: make(chan struct{}),
		subscriber:   subscriber,
	}

	t.subscriptionsMu.Lock()
	t.subscriptionsMu.subscriptions = append(t.subscriptionsMu.subscriptions, s)
	t.subscriptionsMu.Unlock()

	go func() {
		defer close(s.terminatedCh)

		for {
			message, ok := <-s.messageCh
			if !ok {
				s.wg.Wait()
				return
			}

			s.wg.Add(1)

			go func() {
				defer s.wg.Done()
				subscriber(message)
			}()
		}
	}()

	return nil
}

type subscription[T any] struct {
	messageCh    chan T
	terminatedCh chan struct{}
	wg           sync.WaitGroup
	subscriber   func(message T)
}

func (t *Topic[T]) stop() {
	t.closedMu.Lock()
	t.closedMu.closed = true
	t.closedMu.Unlock()

	t.publishWg.Wait()

	close(t.publishCh)

	var wg sync.WaitGroup

	t.subscriptionsMu.RLock()
	for _, s := range t.subscriptionsMu.subscriptions {
		close(s.messageCh)

		wg.Add(1)
		go func(s *subscription[T]) {
			defer wg.Done()
			<-s.terminatedCh
		}(s)
	}
	t.subscriptionsMu.RUnlock()

	wg.Wait()
	<-t.closedCh
}
