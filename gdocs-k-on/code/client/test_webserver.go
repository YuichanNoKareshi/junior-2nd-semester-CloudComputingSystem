package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"sync"
	"time"
)

type websocketClientManager struct {
	conn        *websocket.Conn
	addr        *string
	path        string
	sendMsgChan chan string
	recvMsgChan chan string
	isAlive     bool
	timeout     int
}

// 构造函数
func NewWsClientManager(addrIp, addrPort, path string, timeout int) *websocketClientManager {
	addrString := addrIp + ":" + addrPort
	var sendChan = make(chan string, 10)
	var recvChan = make(chan string, 10)
	var conn *websocket.Conn
	return &websocketClientManager{
		addr:        &addrString,
		path:        path,
		conn:        conn,
		sendMsgChan: sendChan,
		recvMsgChan: recvChan,
		isAlive:     false,
		timeout:     timeout,
	}
}

// 链接服务端
func (wsc *websocketClientManager) dail() {
	var err error
	u := url.URL{Scheme: "ws", Host: *wsc.addr, Path: wsc.path}
	log.Printf("connecting to %s", u.String())
	wsc.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return

	}
	wsc.isAlive = true
	log.Printf("connecting to %s 链接成功！！！", u.String())

}

// 发送消息
func (wsc *websocketClientManager) sendMsgThread() {
	go func() {
		for i := 0; i < 1; i++ {
			// wsc.sendMsgChan <- "test"
			msg := <-wsc.sendMsgChan
			err := wsc.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Println("write:", err)
				continue
			}
		}
	}()
}

// 读取消息
func (wsc *websocketClientManager) readMsgThread() {
	go func() {
		for {
			if wsc.conn != nil {
				_, message, err := wsc.conn.ReadMessage()
				if err != nil {
					log.Println("read:", err)
					wsc.isAlive = false
					// 出现错误，退出读取，尝试重连
					break
				}
				log.Printf("recv: %s", message)
				// 需要读取数据，不然会阻塞
				wsc.recvMsgChan <- string(message)
			}

		}
	}()
}

// 开启服务进行测试
func (wsc *websocketClientManager) start() {
	for {
		if wsc.isAlive == false {
			wsc.dail()
			// testCreate(wsc)
			// testOpen(wsc)
			// testEdit(wsc)
			// testUpdate(wsc)
			// testMtt(wsc)
			// testRecover(wsc)
			// testCt(wsc)
			testRollBack(wsc)
			// testLog(wsc)
			wsc.readMsgThread()
		}
		time.Sleep(time.Second * time.Duration(wsc.timeout))
	}
}

func main() {
	wsc := NewWsClientManager("localhost", "8080", "/ws", 100)
	wsc.start()
	var w1 sync.WaitGroup
	w1.Add(1)
	w1.Wait()
}

type msgInfo struct {
	Type     string `json:"type"`
	Row      uint64 `json:"row"`
	Column   uint64 `json:"column"`
	NewValue string `json:"newValue"`
	FileName string `json:"fileName"`
	UserName string `json:"userName"`
	// OldValue string `json:"oldValue"`
}

func testCreate(wsc *websocketClientManager) {
	createMsg1 := msgInfo{"create", 1, 1, "newvalue", "file2", "xzl"}
	createMsg2 := msgInfo{"create", 1, 1, "newvalue", "file2", "xzl"}
	b1, _ := json.Marshal(createMsg1)
	b2, _ := json.Marshal(createMsg2)
	// 正常测试
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in create test:", err1)
	}
	// 创建重复文件
	err2 := wsc.conn.WriteMessage(1, b2)
	if err2 != nil {
		fmt.Println("error in create test:", err2)
	}
}

func testOpen(wsc *websocketClientManager) {
	openMsg := msgInfo{"open", 1, 1, "newvalue", "file2", "zyt"}
	b1, _ := json.Marshal(openMsg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in open file:", err1)
	}
}

func testEdit(wsc *websocketClientManager) {
	editMsg := msgInfo{"editing", 1, 1, "data1", "file2", "zyt"}
	b1, _ := json.Marshal(editMsg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in edit file:", err1)
	}
}

func testUpdate(wsc *websocketClientManager) {
	updateMsg := msgInfo{"update", 1, 1, "data1", "file2", "zyt"}
	b1, _ := json.Marshal(updateMsg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in update file", err1)
	}
}

func testMtt(wsc *websocketClientManager) {
	mttMsg := msgInfo{"mtt", 1, 1, "updatevalue", "file1", "xzl"}
	b1, _ := json.Marshal(mttMsg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in mtt file", err1)
	}
}

func testRecover(wsc *websocketClientManager) {
	msg := msgInfo{"recover", 1, 1, "updatevalue", "file1", "xzl"}
	b1, _ := json.Marshal(msg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in recover file:", err1)
	}

}

func testCt(wsc *websocketClientManager) {
	msg := msgInfo{"ct", 1, 1, "updatevalue", "file1", "xzl"}
	b1, _ := json.Marshal(msg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in clear trash", err1)
	}
}

func testRollBack(wsc *websocketClientManager) {
	msg := msgInfo{"rollback", 1, 1, "updatevalue", "filexzl", "xzl"}
	b1, _ := json.Marshal(msg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in rollback", err1)
	}
}

func testLog(wsc *websocketClientManager) {
	msg := msgInfo{"log", 1, 1, "", "", "xzl"}
	b1, _ := json.Marshal(msg)
	err1 := wsc.conn.WriteMessage(1, b1)
	if err1 != nil {
		fmt.Println("error in get log", err1)
	}
}
