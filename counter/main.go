package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/AxelUser/maelstrom-walkthrough/internal/crdt"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type addReq struct {
	Delta int `json:"delta"`
}

type syncReq struct {
	Value int `json:"value"`
}

type counter struct {
	n      *maelstrom.Node
	s      map[string]*crdt.Accumulator[int, int]
	initWG sync.WaitGroup
}

func createCounter(n *maelstrom.Node) *counter {
	s := counter{
		n: n,
		s: map[string]*crdt.Accumulator[int, int]{},
	}
	s.initWG.Add(1)
	return &s
}

func (gs *counter) init() {
	for _, n := range gs.n.NodeIDs() {
		gs.s[n] = crdt.CreateAccumulator(0, func(acc int, new int) int {
			return acc + new
		}, func(cur int) int {
			return cur
		})
	}
	gs.initWG.Done()
}

func (gs *counter) startSync(cd time.Duration) {
	if gs.n.NodeIDs() == nil {
		return
	}

	gs.init()

	go func() {
		for {
			value := gs.s[gs.n.ID()].Get()
			for _, dst := range gs.n.NodeIDs() {
				if dst == gs.n.ID() {
					continue
				}
				gs.n.Send(dst, map[string]any{
					"type":  "sync",
					"value": value,
				})
			}
			time.Sleep(cd)
		}
	}()
}

func (gs *counter) add(element int) {
	gs.initWG.Wait()
	gs.s[gs.n.ID()].Add(element)
}

func (gs *counter) read() int {
	gs.initWG.Wait()
	sum := 0

	for _, acc := range gs.s {
		sum += acc.Get()
	}

	return sum
}

func (gs *counter) sync(src string, element int) {
	gs.s[src].Set(element)
}

func main() {
	n := maelstrom.NewNode()
	gs := createCounter(n)

	n.Handle("init", func(msg maelstrom.Message) error {
		gs.startSync(time.Second * 5)
		return nil
	})

	n.Handle("add", func(msg maelstrom.Message) error {
		var req addReq
		err := json.Unmarshal(msg.Body, &req)
		if err != nil {
			return err
		}

		gs.add(req.Delta)

		return n.Reply(msg, map[string]string{
			"type": "add_ok",
		})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		return n.Reply(msg, map[string]any{
			"type":  "read_ok",
			"value": gs.read(),
		})
	})

	n.Handle("sync", func(msg maelstrom.Message) error {
		var req syncReq
		err := json.Unmarshal(msg.Body, &req)
		if err != nil {
			return err
		}

		gs.sync(msg.Src, req.Value)
		return nil
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
