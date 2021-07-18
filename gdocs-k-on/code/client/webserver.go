// +build ignore
// 希望前端发送的line与column都以1开始，便于文件内容存放位置的计算
// config 结构为：
// configFileNum(uint64) totalLen(uint64) filenameLen(uint64) filename filesize(uint64) validbit......
// 当一个用户登入时，需要传输当前正在编辑的文件名，将文件名与用户进行关联，便于消息传回时用户的识别

// delete
// 对于将文件放入回收站的功能，在config中给每个文件添加validbit字段
// 在读出文件列表时，忽略回收站中的文件
// 在执行回收站操作时，将文件从fileMap中移除并将config中的对应字段修改成0

package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-zookeeper/zk"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	// "strings"
	// "time"
)

// 文件结构
type file struct {
	FileName string
	FileSize uint64
}

// fileName -> fileSize
var fileMap map[string]uint64

// fileName -> fileSize
var trashMap map[string]uint64

// 文件名与用户链接信息的 Map
// 以文件名作为索引，value 为当前所有参与编辑文件的用户 slice
// 当用于第一次链接时需要传递链接的 fileName 用以保存
var connMap map[string]([]*websocket.Conn)

// 为每一个链接创建一个node
var nodeMap map[*websocket.Conn](*zk.Conn)

// 记录锁的拥有者
// key: lockIndex value: owner
var lockMap map[string]string

// 每个单元格的大小
var blockSize uint64 = 80

// 列数（暂时为固定值）
var colNum uint64 = 60

// 默认行数
var rowNum uint64 = 84

// 所有链接的记录，包括非文件编辑链接
// 在 connect 时将链接加入，在 close 时将链接移除
var allConnMap map[*websocket.Conn]int

// 我们需要定义一个 Upgrader
// 它需要定义 ReadBufferSize 和 WriteBufferSize
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// 可以用来检查连接的来源
	// 这将允许从我们的 React 服务向这里发出请求。
	// 现在，我们可以不需要检查并运行任何连接
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --------- handlers begin ----------------

// -------------  handlers end ----------------

// 定义一个 reader 用来监听往 WS 发送的新消息
func reader(conn *websocket.Conn) {
	for {
		fmt.Println("reader")
		fmt.Println("[READER] show the connMap")
		for k, _ := range connMap {
			fmt.Println("Filename", k, "num of conn:", len(connMap[k]))
		}
		fmt.Println("[ALLCONNMAP] the size is:", len(allConnMap))
		fmt.Println("[ALLCONNMAP]", allConnMap)

		// 读消息
		messageType, p, err := conn.ReadMessage()
		if err != nil { // conn close的时候会到这里！！！
			removeConn(conn)
			fmt.Println("[READER] service stop for a connection!")
			log.Println(err)
			return
		}
		// 打印消息
		fmt.Print("receive: ")
		fmt.Println(string(p))

		// 处理收到的 json 信息
		// 解析为 receiveData
		var receiveData msgInfo
		json.Unmarshal([]byte(p), &receiveData)

		// 检查处理后的结果
		fmt.Println("[reader] Check the parse result")
		fmt.Println(receiveData)

		handleType := receiveData.Type
		// messageValue := receiveData.NewValue
		fileName := receiveData.FileName

		userName := receiveData.UserName

		// 添加一个新的 Type 作为第一次链接的信息，前端传递链接信息与 fileName，后端存储对应信息
		// 暂时没用，直接在create与open时掉用handleConnect即可
		if handleType == "connect" {
			// 将新加入的链接记录下来
			commonConnect(conn)
			sendBakcMsg(conn)
		}

		// open 打开文件
		if handleType == "open" {
			createConnect(fileName, conn)
			handleOpen(fileName, conn, messageType, userName)
		}

		// create 创建文件
		if handleType == "create" {
			handleCreate(fileName, conn, userName)
		}

		// 用户选中格子希望进行编辑
		if handleType == "editing" {
			handleEdit(receiveData, messageType, p, conn)
		}

		// update 修改文件单元格，并将修改结果转发给所有参与对应文件链接的前端
		if handleType == "update" {
			handleUpdate(receiveData, messageType, p, conn)
		}

		// close 当前用户停止对文件的编辑，将用户conn从connMap中移除
		if handleType == "close" {
			handleClose(fileName, conn)
		}

		// 将文件移动到回收站里，并将更新后的 fileMap(现有文件列表) 返回给前端
		if handleType == "mtt" {
			moveToTrash(fileName, conn, userName)
		}

		// 从回收站中恢复文件，返回type为RECOVERET
		if handleType == "recover" {
			recoverWithRet(fileName, userName, conn)
			// recoverFile(fileName, userName)
		}

		// 从回收站中彻底删除文件，返回type为CTRET
		if handleType == "ct" {
			clearTrash(fileName, userName, conn)
		}

		// 获取回收站文件列表
		if handleType == "trashList" {
			getTrashList(conn)
		}

		// 获取 log
		if handleType == "log" {
			showLog(conn)
		}

		// 回滚
		if handleType == "rollback" {
			handleRollBack(userName, fileName, conn)
		}
	}
}

// 定义 WebSocket 服务处理函数
func serveWs(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Host)

	// 将连接更新为 WebSocket 连接
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// 一直监听 WebSocket 连接上传来的新消息
	// 启动一个新协程
	reader(ws)
}

func setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Simple Server")
	})

	// 将 `/ws` 端点交给 `serveWs` 函数处理
	http.HandleFunc("/ws", serveWs)
}

func main() {
	// 初始化 fileMap
	fileMap = make(map[string]uint64)
	// 初始化 connMap
	connMap = make(map[string]([]*websocket.Conn))
	// 初始化 trashMap
	trashMap = make(map[string]uint64)
	// 初始化 nodeMap
	nodeMap = make(map[*websocket.Conn](*zk.Conn))
	// 初始化 lockMap
	lockMap = make(map[string]string)
	// 初始化 allConnMap
	allConnMap = make(map[*websocket.Conn]int)

	// 将现有的文件信息存储在config中
	// config结构为configFileNum(uint64) totalLen(uint64) filenameLen(uint64) filename filesize(uint64) validBit......

	// 尝试创建config
	noConfig := create("config")
	if noConfig {
		fmt.Println("[MAIN] Start init the server by creating the config file!")
		// 若第一次启动后端，则创建成功，写入以下信息
		cfn_str := numExtend(0, 8)
		tl_str := numExtend(16, 8)
		write("config", 0, cfn_str+tl_str)
	} else {
		// 若创建失败，说明已经存在config,直接读取构建fileMap即可
		// 读取 config
		fmt.Println("[config] Start get perior message for server!")
		var offset uint64 = 0

		fileNumStr := read("config", offset, 8)
		offset += 8

		// totalLenStr := read("config", offset, 8)
		offset += 8

		fileNum, _ := strconv.Atoi(fileNumStr)
		// 根据 config 构建fileMap
		for i := 0; i < fileNum; i++ {

			currFileNameLen, _ := strconv.Atoi(read("config", offset, 8))
			offset += 8

			currFileName := read("config", offset, uint64(currFileNameLen))
			offset += uint64(currFileNameLen)

			currFileSize, _ := strconv.Atoi(read("config", offset, 8))
			offset += 8

			currFileValidBit, _ := strconv.Atoi(read("config", offset, 1))
			offset += 1

			fmt.Println("[BUILD FILEMAP] FileName is : ", currFileName)
			fmt.Println("[BUILD FILEMAP] FileSize is : ", currFileSize)
			fmt.Println("[BUILD FILEMAP] FileValidBit is :", currFileValidBit)

			// 文件存在时将其加入 fileMap
			// 不存在时将其加入 trashMap
			if currFileValidBit == 1 {
				fileMap[currFileName] = uint64(currFileSize)
			} else {
				trashMap[currFileName] = uint64(currFileSize)
			}
		}
	}

	// 创建记录文件log
	onLog := create("gDocLog")
	if onLog {
		// 最开始存储当前文件的大小
		fmt.Println("[MAIN] create the gDocLog file")
		write("gDocLog", 0, numExtend(9, 8))
		write("gDocLog", 8, "\t")
	} else {
		fmt.Println("[MAIN] gDocLog already exists")
	}

	fmt.Println(fileMap)
	fmt.Println(trashMap)

	fmt.Println("webserver start!")
	setupRoutes()
	http.ListenAndServe(":8080", nil)
}
