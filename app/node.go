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

//节点的状态代码
const IDLE = 0
const Active = 1
const Expired = 2

//节点结构体
type Node struct {
	//节点的id
	ID string `json:"id"`
	//节点继续通讯的概率
	HP float64 `json:"hp"`
	//节点的状态
	State int `json:"state"`
	//互斥锁，保证关键代码只被一个goroutinue执行
	lock sync.Mutex
	//节点的数据
	Data float64 `json:"data"`
	//消息管道，当节点被激活时，会不断从中读取数据
	Messages chan *Message
	//消息的输出通道，消息将会通过websocket传递给前端
	Output chan *Event
	//保存通讯记录，以便结束后统计
	Records []*Record
	//上下文，用来管理goroutinue，防止goroutine泄漏
	context context.Context
	//上下文取消方法
	Cancel context.CancelFunc
	//节点的配置文件
	Config *model.NodeConfig `json:"config"`
	//全局状态
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
		Records:     make([]*Record, 0),
		context:     ctx,
		Cancel:      cancel,
		Config:      config,
		GlobalState: global,
	}
	go n.handleMessage()
	return n
}

//激活节点的方法
func (node *Node) Active() {
	if node.State == Active {
		return
	}
	node.HP = 1
	node.State = Active
	node.sendChangeStateEvent()
	go node.syncData()
}

//处理节点消息的方法
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
			//防止重复接受同一个节点的消息
			length := len(node.Records)
			switch length {
			case 0:
				break
			case 1:
				if node.Records[0].Sender == message.From {
					node.Messages <- message
					continue
				}
			default:
				if node.Records[length-1].Sender == message.From || node.Records[length-2].Sender == message.From {
					node.Messages <- message
					continue
				}
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
					result := (message.Data + node.Data) / 2
					response.Code = Updated
					response.Data = result
					node.Data = result
					node.addRecord(message.From, node.Data)
					node.sendUpdateEvent()
				} else {
					response.Code = RejectUpdate
				}
				node.lock.Unlock()
			}
			go func(res Response) {
				message.Callback <- &res
			}(response)
		}
	}
}

//添加记录的方法
func (node *Node) addRecord(sender string, data float64) {
	node.Records = append(node.Records, &Record{
		Sender:   sender,
		Receiver: node.ID,
		Data:     data,
	})
}

//处理节点返回
func (node *Node) handleResponse(chancel chan *Response) {
	response := <-chancel
	switch response.Code {
	case RejectUpdate:
		node.HP = node.HP * node.Config.Decay
	case Updated:
		node.Data = response.Data
		node.sendUpdateEvent()
	}
}

//同步数据
func (node *Node) syncData() {
	count := 0
	for {
		//判断是否继续发送消息
		if rand.Float64() > node.HP || node.Config.MaxRound <= count {
			node.State = Expired
			node.sendChangeStateEvent()
			return
		}
		select {
		case <-node.context.Done():
			return
		default:
			callback := make(chan *Response)
			node.lock.Lock()
			message := &Message{
				Data:     node.Data,
				From:     node.ID,
				Callback: callback,
			}
			node.lock.Unlock()
			//通过管道传送数据
			node.Messages <- message
			node.handleResponse(callback)
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

//重制节点状态，其中会重新创建上下文，停止当前的消息同步，重新生成节点数值
func (node *Node) Reset() {
	node.Stop()
	node.Records = make([]*Record, 0)
	node.Data = rand.Float64()*(node.Config.Max-node.Config.Min) + node.Config.Min
	ctx, cancel := context.WithCancel(context.Background())
	node.context = ctx
	node.Cancel = cancel
	go node.handleMessage()
}

//停止节点当前的同步
func (node *Node) Stop() {
	node.Cancel()
	node.State = IDLE
}

func (node *Node) Log(message ...interface{}) {
	log.Println(node.ID, ":", message)
}
