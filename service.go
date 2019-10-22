package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"gossip/model"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var server = gin.New()

var node = server.Group("/node")
var statistic = server.Group("/statistic")
var event = server.Group("/event")
var GlobalState = &atomic.Value{}
var connections = sync.Map{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func init() {
	GlobalState.Store(false)

	path, e := filepath.Abs(filepath.Dir(os.Args[0]))
	if e != nil {
		log.Panicln(e)
	}
	server.Static("/js", path+"/dist/js")
	server.Static("/css", path+"/dist/css")
	server.Static("/fonts", path+"/dist/fonts")
	server.LoadHTMLFiles(path + "/dist/index.html")
	server.GET("/", func(ctx *gin.Context) {
		ctx.HTML(200, "index.html", nil)
	})

	node.GET("/all", getAllNode)
	node.POST("/add/:amount", addNode)
	node.POST("/delete/:id", deleteNode)
	node.GET("/start", start)
	node.GET("/reset", reset)
	node.GET("/clear", clear)

	statistic.GET("/mean", getMean)
	statistic.GET("/decay", getDecay)
	statistic.GET("/value/max", getMaxValue)
	statistic.GET("/value/min", getMinValue)
	statistic.GET("/round/max", getMaxRound)
	statistic.GET("/round/min", getMinRound)
	statistic.GET("/round/mean", getMeanRound)
	statistic.GET("/mse/:correct", getMse)
	statistic.GET("/save/:correct", save)
	statistic.GET("/record/byNumber", getRecordsByNumber)
	statistic.GET("/record/byDecay", getRecordsByDecay)
	statistic.GET("/record/all", getAllRecords)
	statistic.POST("/export", export)

	event.GET("/connect", connectEvent)
}

func export(ctx *gin.Context) {
	var data struct {
		Headers []string   `json:"headers"`
		Data    [][]string `json:"data"`
	}
	err := ctx.BindJSON(&data)
	if err != nil {
		ctx.String(400, "error")
		return
	}
	response := ""
	response = response + strings.Join(data.Headers, ",") + "\n"
	for _, v := range data.Data {
		response = response + strings.Join(v, ",") + "\n"
	}
	response = strings.TrimSpace(response)
	ioutil.WriteFile("data.csv", []byte(response), os.ModePerm)
	ctx.String(200, "success")
}

func getMeanRound(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%.2f", GetMeanRound()))
}

func getAllRecords(ctx *gin.Context) {
	ctx.JSON(200, GetAllRecords())
}

func getDecay(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%.2f", GetDecay()))
}

func getRecordsByNumber(ctx *gin.Context) {
	ctx.JSON(200, GetRecordsByNodeAmount())
}

func getRecordsByDecay(ctx *gin.Context) {
	ctx.JSON(200, GetRecordsByDecay())
}

func save(ctx *gin.Context) {
	s, b := ctx.Params.Get("correct")
	if !b {
		ctx.String(400, "")
		return
	}
	float, e := strconv.ParseFloat(s, 64)
	if e != nil {
		ctx.String(400, "")
		return
	}
	SaveRecord(float)
	ctx.String(200, "success")
}

func getMse(ctx *gin.Context) {
	s, b := ctx.Params.Get("correct")
	if !b {
		ctx.String(400, "")
		return
	}
	float, e := strconv.ParseFloat(s, 64)
	if e != nil {
		ctx.String(400, "")
		return
	}
	ctx.String(200, fmt.Sprintf("%.2f", GetMSE(float)))
}

func getMinRound(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%d", GetMinRound()))
}

func getMaxRound(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%d", GetMaxRound()))
}

func getMinValue(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%.2f", GetMinValue()))
}

func getMaxValue(ctx *gin.Context) {
	ctx.String(200, fmt.Sprintf("%.2f", GetMaxValue()))
}

func reset(context *gin.Context) {
	GlobalState.Store(false)
	ResetAll()
	context.String(200, "success")
}

func getAllNode(ctx *gin.Context) {
	ctx.JSON(200, GetAllNode())
}

func getMean(ctx *gin.Context) {
	ctx.JSON(200, fmt.Sprintf("%.2f", GetMean()))
}

func addNode(ctx *gin.Context) {
	var config model.NodeConfig
	amount := ctx.Param("amount")
	i, _ := strconv.Atoi(amount)
	_ = ctx.BindJSON(&config)
	AddNodeBatch(&config, i)
	ctx.String(200, "success")
}

func deleteNode(ctx *gin.Context) {
	id := ctx.GetString("id")
	DeleteNode(id)
	ctx.String(200, "success")
}

func start(ctx *gin.Context) {
	GlobalState.Store(true)
	Active()
	ctx.String(200, "success")
}

func connectEvent(ctx *gin.Context) {
	log.Println("connected")
	c, e := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if e != nil {
		log.Println(e)
		return
	}

	id := uuid.NewV4().String()
	connection := &connection{
		id:   id,
		lock: sync.Mutex{},
		c:    c,
	}
	connections.Store(id, connection)
	ticker := time.NewTicker(time.Second)
	defer func() {
		connections.Delete(id)
		ticker.Stop()
	}()
	context, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			if _, _, err := connection.c.NextReader(); err != nil {
				cancel()
				connections.Delete(id)
				connection.c.Close()
				break
			}
		}
	}()
	for {
		select {
		case <-ticker.C:
			connection.lock.Lock()
			_ = connection.c.SetWriteDeadline(time.Now().Add(time.Second))
			e := connection.c.WriteMessage(websocket.PingMessage, nil)
			connection.lock.Unlock()
			if e != nil {
				break
			}
		case <-context.Done():
			return
		}
	}
}

func clear(ctx *gin.Context) {
	GlobalState.Store(false)
	Clear()
}
