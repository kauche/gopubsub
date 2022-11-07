# gopubsub

`gopubsub` is a generic [topic-based Publish–Subscribe](https://en.wikipedia.org/wiki/Publish%E2%80%93subscribe_pattern) library for Go backed by Goroutines and Channels.

```go
package main

import (
	"context"

	"github.com/kauche/gopubsub"
)

type greetingMessage struct {
	greeting string
}

func main() {
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
		println(err)
		return
	}

	// Subscribe the topic. This call of Subscribe is non-blocking.
	// The function passed to Subscribe is called when a message is published to the topic.
	err := topic.Subscribe(func(message greetingMessage) {
		println(message.greeting)
	})
	if err != nil {
		println(err)
		return
	}

	cancel()
	<-terminated
}
```

See https://pkg.go.dev/github.com/kauche/gopubsub for more details.
