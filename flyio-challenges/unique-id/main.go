package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	if n == nil {
		log.Fatalln("unable to create a maelstrom node")
	}

	n.Handle("generate", func(m maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(m.Body, &body); err != nil {
			return err
		}

		body["type"] = "generate_ok"
		body["id"] = fmt.Sprintf("%d%s", time.Now().UnixNano(), n.ID())

		return n.Reply(m, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
