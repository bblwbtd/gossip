package app

import (
	"context"
	"fmt"
	"github.com/satori/go.uuid"
	"gossip/model"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const IDLE = 0
const Active = 1
const Expired = 2

type Node struct {
	ID          string  `json:"id"`
	HP          float64 `json:"hp"`
	State       int     `json:"state"`
	lock        sync.Mutex
	Data        float64 `json:"data"`
	Messages    chan *Message
	Output      chan *Event
	Responses   chan *Response
	Records     []*Record
	context     context.Context
	Cancel      context.CancelFunc
	Config      *model.NodeConfig `json:"config"`
	GlobalState *atomic.Value
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewNode(config *model.NodeConfig, messageChannel chan *Message, outPutChannel chan *Event, global *atomic.Value) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		ID:          uuid.NewV4().String(),
		lock:        sync.Mutex{},
		Data:        rand.Float64()*(config.Max-config.Min) + config.Min,
		Messages:    messageChannel,
		Output:      outPutChannel,
		Responses:   make(chan *Response, 100),
		Records:     make([]*Record, 0),
		context:     ctx,
		Cancel:      cancel,
		Config:      config,
		GlobalState: global,
	}
	go n.handleMessage()
	return n
}

func (node *Node) Active() {
	if node.State == Active {
		return
	}
	node.HP = 1
	node.State = Active
	node.sendChangeStateEvent()
	go node.handleResponse()
	go node.syncData()
}

func (node *Node) handleMessage() {
	defer func() {
		recover()
	}()
	for {
		select {
		case _ = <-node.context.Done():
			return
		case message, ok := <-node.Messages:
			if !ok {
				return
			} else if !node.GlobalState.Load().(bool) {
				continue
			}
			var response Response
			if node.State == Expired {
				response.Code = RejectUpdate
			} else {
				if node.State == IDLE {
					node.Active()
				}
				node.lock.Lock()
				if message.From == node.ID {
					node.lock.Unlock()
					node.Messages <- message
					continue
				} else if message.Data != node.Data {
					node.Data = (message.Data + node.Data) / 2
					response.Code = Updated
					response.Data = node.Data
					node.sendUpdateEvent()
				} else {
					response.Code = RejectUpdate
				}
				node.addRecord(message.From, node.Data)
				node.lock.Unlock()
			}
			message.Callback(&response)
		}
	}
}

func (node *Node) addRecord(sender string, data float64) {
	node.Records = append(node.Records, &Record{
		Sender:   sender,
		Receiver: node.ID,
		Data:     data,
	})
}

func (node *Node) handleResponse() func(response *Response) {
	return func(response *Response) {
		switch response.Code {
		case RejectUpdate:
			node.HP = node.HP * node.Config.Decay
		case Updated:
			node.Data = response.Data
			node.sendUpdateEvent()
		}
	}
}

func (node *Node) syncData() {
	count := 0
	for {
		value := rand.Float64()
		if value > node.HP || node.Config.MaxRound <= count {
			node.State = Expired
			node.sendChangeStateEvent()
			fmt.Println(len(node.Records))
			return
		}
		select {
		case <-node.context.Done():
			return
		default:
			node.lock.Lock()
			message := &Message{
				Data:     node.Data,
				From:     node.ID,
				Callback: node.handleResponse(),
			}
			node.lock.Unlock()
			node.Messages <- message
		}
		for i := 0; i < node.Config.Delay; i++ {
			time.Sleep(time.Millisecond)
		}
		count++
	}
}

func (node *Node) sendChangeStateEvent() {
	node.sendEvent(ChangeState, node.State)
}

func (node *Node) sendUpdateEvent() {
	node.sendEvent(UpdateValue, fmt.Sprintf("%.2f", node.Data))
}

func (node *Node) sendEvent(Type int, data interface{}) {
	node.Output <- &Event{
		From: node.ID,
		Type: Type,
		Data: data,
	}
}

func (node *Node) Reset() {
	node.Stop()
	node.Records = make([]*Record, 0)
	node.Data = rand.Float64()*(node.Config.Max-node.Config.Min) + node.Config.Min
	ctx, cancel := context.WithCancel(context.Background())
	node.context = ctx
	node.Cancel = cancel
	go node.handleMessage()
}

func (node *Node) Stop() {
	node.Cancel()
	node.State = IDLE
}

func (node *Node) Log(message ...interface{}) {
	log.Println(node.ID, ":", message)
}
