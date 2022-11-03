package gopubsub_test

import (
	"fmt"

	"github.com/kauche/gopubsub"
)

type greetingMessage struct {
	greeting string
}

func Example() {
	// Create a topic with a message type whatever you want to publish and subscribe.
	topic, stop := gopubsub.NewTopic[greetingMessage]()
	defer stop() // At the end, call stop() to stop the topic and drain all published messages for the topic.

	// Publish a message to the topic. This call of Publish is non-blocking.
	topic.Publish(greetingMessage{greeting: "Hello, gopubsub!"})

	// Subscribe the topic. This call of Subscribe is non-blocking.
	// The function passed to Subscribe is called when a message is published to the topic.
	topic.Subscribe(func(message greetingMessage) {
		fmt.Println(message.greeting)
	})
}
