package gopubsub_test

import (
	"context"

	"github.com/kauche/gopubsub"
)

type greetingMessage struct {
	greeting string
}

func Example() {
	// Create a topic with a type which you want to publish and subscribe.
	topic := gopubsub.NewTopic[greetingMessage]()

	ctx, cancel := context.WithCancel(context.Background())
	terminated := make(chan struct{})

	go func() {
		// Start the topic. This call of Start blocks until the context is canceled.
		if err := topic.Start(ctx); err != nil {
			println(err)
			return
		}

		terminated <- struct{}{}
	}()

	// Publish a message to the topic. This call of Publish is non-blocking.
	if err := topic.Publish(greetingMessage{greeting: "Hello, gopubsub!"}); err != nil {
		cancel()
		println(err)
		return
	}

	// Subscribe the topic. This call of Subscribe is non-blocking.
	// The function passed to Subscribe is called when a message is published to the topic.
	err := topic.Subscribe(func(message greetingMessage) {
		println(message.greeting)
	})
	if err != nil {
		cancel()
		println(err)
		return
	}

	cancel()
	<-terminated
}
