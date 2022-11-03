package gopubsub

import (
	"sync"
)

const (
	defaultPublishChannelBufferSize   = 100
	defaultSubscribeChannelBufferSize = 100
)

// Topic is a publish-subscribe topic.
type Topic[T any] struct {
	doneCh    chan struct{}
	publishCh chan T

	publishWg sync.WaitGroup

	subscriptionsMu struct {
		sync.RWMutex
		subscriptions []*subscription[T]
	}

	stoppedMu struct {
		sync.RWMutex
		stopped bool
	}
}

// NewTopic creates a new topic. The returned topic is stopped and drains all published messages when the returned stop function is called.
func NewTopic[T any]() (*Topic[T], StopFunc) {
	t := &Topic[T]{
		doneCh:    make(chan struct{}),
		publishCh: make(chan T, defaultPublishChannelBufferSize), // TODO: The buffer size should be configurable.
	}

	go func() {
		defer close(t.doneCh)

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

	return t, t.stop
}

// Publish publishes a message to the topic. This method is non-blocking and concurrent-safe.
func (t *Topic[T]) Publish(message T) {
	t.stoppedMu.RLock()
	defer t.stoppedMu.RUnlock() // In order to avoid the race condition between Topic.Publish and Topic.stop, defer is necessary.
	if t.stoppedMu.stopped {
		return
	}

	t.publishWg.Add(1)
	t.publishCh <- message
}

// Subscribe registers the passed function as a subscriber to the topic. This method is non-blocking and concurrent-safe.
// The function passed to Subscribe is called when a message is published to the topic.
func (t *Topic[T]) Subscribe(subscriber func(message T)) {
	t.stoppedMu.RLock()
	if t.stoppedMu.stopped {
		return
	}
	t.stoppedMu.RUnlock()

	s := &subscription[T]{
		messageCh:  make(chan T, defaultSubscribeChannelBufferSize), // TODO: The buffer size should be configurable.
		doneCh:     make(chan struct{}),
		subscriber: subscriber,
	}

	t.subscriptionsMu.Lock()
	t.subscriptionsMu.subscriptions = append(t.subscriptionsMu.subscriptions, s)
	t.subscriptionsMu.Unlock()

	go func() {
		defer close(s.doneCh)

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
}

// StopFunc tells the topic to stop and drain all published messages.
type StopFunc func()

type subscription[T any] struct {
	messageCh  chan T
	doneCh     chan struct{}
	wg         sync.WaitGroup
	subscriber func(message T)
}

func (t *Topic[T]) stop() {
	t.stoppedMu.Lock()
	t.stoppedMu.stopped = true
	t.stoppedMu.Unlock()

	t.publishWg.Wait()

	close(t.publishCh)

	var wg sync.WaitGroup

	t.subscriptionsMu.RLock()
	for _, s := range t.subscriptionsMu.subscriptions {
		close(s.messageCh)

		wg.Add(1)
		go func(s *subscription[T]) {
			defer wg.Done()
			<-s.doneCh
		}(s)
	}
	t.subscriptionsMu.RUnlock()

	wg.Wait()
	<-t.doneCh
}
