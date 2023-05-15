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
	Element int `json:"element"`
}

type syncReq struct {
	Elements []int `json:"elements"`
}

type gSet struct {
	n      *maelstrom.Node
	s      map[string]*crdt.Accumulator[map[int]struct{}, int]
	initWG sync.WaitGroup
}

func createGSet(n *maelstrom.Node) *gSet {
	s := gSet{
		n: n,
		s: map[string]*crdt.Accumulator[map[int]struct{}, int]{},
	}
	s.initWG.Add(1)
	return &s
}

func (gs *gSet) init() {
	for _, n := range gs.n.NodeIDs() {
		gs.s[n] = crdt.CreateAccumulator(map[int]struct{}{}, func(acc map[int]struct{}, new int) map[int]struct{} {
			acc[new] = struct{}{}
			return acc
		}, func(cur map[int]struct{}) map[int]struct{} {
			cp := map[int]struct{}{}
			for v := range cur {
				cp[v] = struct{}{}
			}
			return cp
		})
	}
	gs.initWG.Done()
}

func (gs *gSet) startSync(cd time.Duration) {
	if gs.n.NodeIDs() == nil {
		return
	}

	gs.init()

	go func() {
		for {
			els := make([]int, 0)
			for el := range gs.s[gs.n.ID()].Get() {
				els = append(els, el)
			}
			for _, dst := range gs.n.NodeIDs() {
				if dst == gs.n.ID() {
					continue
				}

				gs.n.Send(dst, map[string]any{
					"type":     "sync",
					"elements": els,
				})
			}
			time.Sleep(cd)
		}
	}()
}

func (gs *gSet) add(element int) {
	gs.initWG.Wait()
	gs.s[gs.n.ID()].Add(element)
}

func (gs *gSet) read() []int {
	gs.initWG.Wait()
	set := map[int]struct{}{}
	ret := make([]int, 0)

	for _, acc := range gs.s {
		for val := range acc.Get() {
			if _, ok := set[val]; !ok {
				set[val] = struct{}{}
				ret = append(ret, val)
			}
		}
	}

	return ret
}

func (gs *gSet) sync(src string, elements []int) {
	set := map[int]struct{}{}
	for _, v := range elements {
		set[v] = struct{}{}
	}
	gs.s[src].Set(set)
}

func main() {
	n := maelstrom.NewNode()
	gs := createGSet(n)

	n.Handle("init", func(msg maelstrom.Message) error {
		gs.startSync(time.Second * 2)
		return nil
	})

	n.Handle("add", func(msg maelstrom.Message) error {
		var req addReq
		err := json.Unmarshal(msg.Body, &req)
		if err != nil {
			return err
		}

		gs.add(req.Element)

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

		gs.sync(msg.Src, req.Elements)
		return nil
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
