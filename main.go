package main

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"gossip/app"
	"gossip/model"
	"gossip/record"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var nodeTable sync.Map
var messageChannel = make(chan *app.Message, 100)
var outputChannel = make(chan *app.Event, 100)

var commands = map[string]string{
	"windows": "cmd /c start",
	"darwin":  "open",
	"linux":   "xdg-open",
}

func main() {
	runtime.GOMAXPROCS(16)
	go BroadcastEvent()
	go func() {
		Open("http://localhost:2333")
	}()
	err := server.Run("0.0.0.0:2333")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		time.Sleep(time.Second)
	}
}

//打开浏览器
func Open(uri string) error {
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	cmd := exec.Command(run, uri)
	return cmd.Start()
}

//添加节点
func AddNode(config *model.NodeConfig) {
	node := app.NewNode(config, messageChannel, outputChannel, GlobalState)
	nodeTable.Store(node.ID, node)
}

//批量添加节点
func AddNodeBatch(config *model.NodeConfig, amount int) {
	for i := 0; i < amount; i++ {
		AddNode(config)
	}
}

func DeleteNode(id string) {
	nodeTable.Delete(id)
}

func GetNode(id string) *app.Node {
	value, ok := nodeTable.Load(id)
	if !ok {
		return nil
	}
	return value.(*app.Node)
}

func GetAllNode() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		list = append(list, map[string]interface{}{
			"id":    node.ID,
			"state": node.State,
			"hp":    node.HP,
			"data":  fmt.Sprintf("%.2f", node.Data),
		})
		return true
	})
	return list
}

func GetRecords(id string) []*app.Record {
	return GetNode(id).Records
}

func Reset(id string) {
	GetNode(id).Reset()
}

func ResetAll() {
	for len(messageChannel) > 0 {
		<-messageChannel
	}
	for len(outputChannel) > 0 {
		<-outputChannel
	}
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		node.Reset()
		return true
	})
}

func Active() {
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		node.Active()
		return false
	})
}

func Stop(id string) {
	value, ok := nodeTable.Load(id)
	if !ok {
		return
	}
	value.(*app.Node).Stop()
}

func Clear() {
	nodeTable.Range(func(key, value interface{}) bool {
		Stop(key.(string))
		return true
	})
	nodeTable = sync.Map{}
}

func BroadcastEvent() {
	for {
		select {
		case event := <-outputChannel:
			log.Println(event)
			connections.Range(func(key, value interface{}) bool {
				conn := value.(*connection)
				_ = conn.c.SetWriteDeadline(time.Now().Add(time.Second))
				conn.lock.Lock()
				_ = conn.c.WriteJSON(event)
				conn.lock.Unlock()
				return true
			})
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func SaveRecord(correct float64) {
	rec := &record.ExperimentRecord{
		Name:       uuid.NewV4().String(),
		MaxValue:   GetMaxValue(),
		MinValue:   GetMinValue(),
		NodeAmount: len(GetAllNode()),
		Decay:      GetDecay(),
		Mse:        GetMSE(correct),
		MeanLost:   correct - GetMean(),
		MeanRound:  GetMeanRound(),
	}
	record.AddRecord(rec)
}
