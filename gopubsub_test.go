package gopubsub_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/goleak"

	"github.com/kauche/gopubsub"
)

const (
	numPublishmentsPerTopic = 200
	numSubscribersPerTopic  = 200
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestTopic(t *testing.T) {
	t.Parallel()

	topic := gopubsub.NewTopic[int]()

	type subscriberResult struct {
		key int
		val int
	}
	subscribersCh := make(chan subscriberResult, numSubscribersPerTopic*numPublishmentsPerTopic)

	var wg sync.WaitGroup
	wg.Add(numSubscribersPerTopic)
	for i := 0; i < numSubscribersPerTopic; i++ {
		go func(i int) {
			defer wg.Done()
			topic.Subscribe(func(message int) {
				subscribersCh <- subscriberResult{
					key: i,
					val: message,
				}
			})
		}(i)
	}
	wg.Wait()

	wg.Add(numPublishmentsPerTopic)
	for i := 0; i < numPublishmentsPerTopic; i++ {
		go func(i int) {
			defer wg.Done()
			topic.Publish(i)
		}(i)
	}

	ctx, cancel := context.WithCancel(context.Background())

	terminatedCh := make(chan struct{})
	go func() {
		topic.Start(ctx)
		terminatedCh <- struct{}{}
	}()
	wg.Wait()

	cancel()
	<-terminatedCh

	close(subscribersCh)

	actual := make(map[int]map[int]struct{})
	for r := range subscribersCh {
		_, ok := actual[r.key]
		if !ok {
			actual[r.key] = make(map[int]struct{})
		}

		actual[r.key][r.val] = struct{}{}
	}

	expected := make(map[int]map[int]struct{})
	for i := 0; i < numSubscribersPerTopic; i++ {
		expected[i] = make(map[int]struct{})
		for j := 0; j < numPublishmentsPerTopic; j++ {
			expected[i][j] = struct{}{}
		}
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("\n(-actual, +expected)\n%s", diff)
	}

	// Ensure that a publishing and a subscribing a stopped topic does not panic.
	topic.Subscribe(func(message int) {})
	topic.Publish(1)
}
