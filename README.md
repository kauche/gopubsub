# gopubsub

`gopubsub` is a generic [topic-based Publish–Subscribe](https://en.wikipedia.org/wiki/Publish%E2%80%93subscribe_pattern) library for Go backed by Goroutines and Channels.

```go
package main

import (
	"context"
	"fmt"

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
		topic.Start(ctx)

		terminated <- struct{}{}
	}()

	// Publish a message to the topic. This call of Publish is non-blocking.
	topic.Publish(greetingMessage{greeting: "Hello, gopubsub!"})

	// Subscribe the topic. This call of Subscribe is non-blocking.
	// The function passed to Subscribe is called when a message is published to the topic.
	topic.Subscribe(func(message greetingMessage) {
		fmt.Println(message.greeting)
	})

	cancel()
	<-terminated
}
```

See https://pkg.go.dev/github.com/kauche/gopubsub for more details.
