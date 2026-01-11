package main

import (
	"encoding/json"
	"errors"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var arr []int

func main() {
	n := maelstrom.NewNode()
	if n == nil {
		log.Fatal("error in craeting node")
		return
	}

	n.Handle("broadcast", func (m maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(m.Body, &body); err != nil {
			return err
		}

		val, ok := body["message"].(float64)
		if !ok {
			return errors.New("invalid value in broadcast message")
		}

		arr = append(arr, int(val))

		reply := map[string]string {
			"type": "broadcast_ok",
		}
		return n.Reply(m, reply)
	})

	n.Handle("read", func (m maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(m.Body, &body); err != nil {
			return err
		}
		
		reply := map[string]any{
			"type": "read_ok",
			"messages": arr,
		}
		return n.Reply(m, reply)
	})

	n.Handle("topology", func (m maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(m.Body, &body); err != nil {
			return err
		}

		reply := map[string]string{
			"type": "topology_ok",
		}

		return n.Reply(m, reply)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
