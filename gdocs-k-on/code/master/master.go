package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var masterLocation string = "127.0.0.1:" // 8095 8096 8097

var isLeader bool // 全局变量，是true就是leader，是false就是replica master

var path string

// 第一个key是filename，第二个key是chunkNum
var fileToChunkMap map[string]map[uint64]ChunkReplicas

var nodeLocation_1 string = "127.0.0.1:8100"

var nodeLocation_2 string = "127.0.0.1:8101"

var nodeLocation_3 string = "127.0.0.1:8102"

var scheduleList []NodeMetadata = []NodeMetadata{{nodeLocation_1, true, 0}, {nodeLocation_2, true, 0}, {nodeLocation_3, true, 0}}

type NodeMetadatas []NodeMetadata

func (metas NodeMetadatas) Len() int           { return len(metas) }
func (metas NodeMetadatas) Less(i, j int) bool { return metas[i].ChunkNum < metas[j].ChunkNum } // 升序
func (metas NodeMetadatas) Swap(i, j int)      { metas[i], metas[j] = metas[j], metas[i] }

var serverToTimerMap map[string]*time.Timer // 记录chunkserver对应的计时器

type NodeMetadata struct {
	NodeLocation string // chunkserver的ip
	Alive        bool   // 是否还活着
	ChunkNum     uint64 // 记录该chunkserver中chunk的个数，用来做loadbalance
}

// chunk 副本
type ChunkReplicas struct {
	Chunk_1 ChunkMetadata
	Chunk_2 ChunkMetadata
}

// chunk 元数据结构
// 只保存最高 Version 的块的元数据
type ChunkMetadata struct {
	Chunkid       string // 哪个文件的第几个chunk，如test-0
	Chunklocation string // 该chunk在哪个chunkserver，如"nodeLocation_1"
	Version       uint64
}

// chunk 管理结构体
type ChunkMasterHandle struct {
}

// chunk 管理请求结构体（来自gfs_api的rpc）
type ChunkMasterRequest struct {
	Filename   string
	Chunkindex uint64
	Write      bool
	Read       bool

	ServerLocation  string // UpdateChunkVersionReplica和创建chunk的时候会用
	ReplicaLocation string // 创建chunk的时候会用，传给replica备份
	// 告诉 master 该chunk已经更新完毕的相关元数据
	Chunkid string // UpdateChunkVersionReplica会用
	Version uint64 // UpdateChunkVersionReplica会用
}

// chunk 管理响应结构体
type ChunkMasterResponse struct {
	Chunkid       string
	Chunklocation string
	Errornum      int // 0 正常 -1 文件已经被创建 -2 对应文件不存在 -3 chunkid相应的chunk不存在 -4 该master不是leader或是leader，类型不对

	Reply             bool
	AllServerLocation []string // 在getChunkLocation时返回所有存活的chunkserver，方便chunkserver广播
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// chunk 位置方法，根据filename和chunkindex找到该file的第chunkindex个chunk在哪个node
func (this *ChunkMasterHandle) GetChunkLocation(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 不是leader
	if !isLeader {
		res.Errornum = -4
		return nil
	}

	//没有该chunk
	if fileToChunkMap[req.Filename] == nil {
		res.Errornum = -2
		return nil
	}
	fmt.Println("filename", req.Filename)
	fmt.Println("chunkindex", req.Chunkindex)
	chunkreplicas, ok := fileToChunkMap[req.Filename][req.Chunkindex]
	if !ok {
		res.Errornum = -3
		return nil
	}
	var chunk ChunkMetadata
	chunk1 := chunkreplicas.Chunk_1
	chunk2 := chunkreplicas.Chunk_2

	var chunkserverIndex1, chunkserverIndex2 int
	for i := range scheduleList {
		if chunk1.Chunklocation == scheduleList[i].NodeLocation {
			chunkserverIndex1 = i
		}
		if chunk2.Chunklocation == scheduleList[i].NodeLocation {
			chunkserverIndex2 = i
		}
	}

	if !scheduleList[chunkserverIndex1].Alive && !scheduleList[chunkserverIndex2].Alive {
		// 两个chunkserver都死掉了
		log.Fatalln("there are no enough chunkservers!")
	} else if chunk1.Version >= chunk2.Version {
		if scheduleList[chunkserverIndex1].Alive { // chunkserver1还活着
			chunk = chunk1
		} else {
			chunk = chunk2
		}
	} else {
		if scheduleList[chunkserverIndex2].Alive { // chunkserver2还活着
			chunk = chunk2
		} else {
			chunk = chunk1
		}
	}

	res.Chunkid = chunk.Chunkid
	res.Chunklocation = chunk.Chunklocation
	res.Errornum = 0
	var allServerLocation []string
	for _, meta := range scheduleList {
		if meta.Alive {
			allServerLocation = append(allServerLocation, meta.NodeLocation)
		}
	}
	res.AllServerLocation = allServerLocation
	fmt.Println(allServerLocation)
	return nil
}

// ChunkMasterRequest{Filename, Chunkindex, true, false}，Filename和Chunkindex有用
// 增加chunk方法
func (this *ChunkMasterHandle) AddChunk(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 不是leader
	if !isLeader {
		res.Errornum = -4
		return nil
	}

	//没有该chunk
	if fileToChunkMap[req.Filename] == nil {
		res.Errornum = -2
		return nil
	}

	Chunkindex := strconv.Itoa(int(req.Chunkindex))

	Chunkid, location, location_replica, Errornum := AddChunkFunc(req, true)
	log.Println("AddAddChunk filename>" + req.Filename + "/Chunkindex>" + Chunkindex + "/" +
		"Location>" + location + "/" +
		"Location_replica>" + location_replica + "/")

	// 向未抢到锁的master发rpc备份数据
	for i := 8095; i <= 8097; i++ {
		aimMaster := "127.0.0.1:" + strconv.Itoa(i)
		if aimMaster == masterLocation {
			continue // 不用广播给自己
		}

		replicaReq := ChunkMasterRequest{req.Filename, req.Chunkindex, true, false, location, location_replica, "", 0}
		var replicaRes ChunkMasterResponse
		conn, err := rpc.DialHTTP("tcp", aimMaster)
		if err != nil {
			// log.Fatalln("dailing error: ", err)
			continue
		}
		defer conn.Close()

		err = conn.Call("ChunkMasterHandle.AddChunkReplica", replicaReq, &replicaRes) // 创建新块
		if err != nil {
			// log.Fatalln("ChunkMasterHandle error: ", err)
			fmt.Println("ChunkMasterHandle error: ", err)
		}
	}

	res.Chunkid = Chunkid
	res.Chunklocation = location
	res.Errornum = Errornum
	return nil
}

// AddChunk和recovery的时候会调用的函数，返回res中的Chunkid、location和Errornum
func AddChunkFunc(req ChunkMasterRequest, willrpc bool) (string, string, string, int) {
funcBegin:
	var Chunkid string = ""
	var ChunkAlive bool = false
	var location string
	var Chunkid_replica string = ""
	var replicaAlive bool = false
	var location_replica string

	var metas []NodeMetadata
	for _, meta := range scheduleList {
		if meta.Alive {
			metas = append(metas, meta)
		}
	}
	sort.Sort(NodeMetadatas(metas)) // 按ChunkNum从小到大排序

	if willrpc && !ChunkAlive {
		location = metas[0].NodeLocation
		Chunkid, ChunkAlive = AddChunk(req.Filename, location, req.Chunkindex)
		if !ChunkAlive {
			goto funcBegin
		}
	} else if !willrpc { // recover，req中含有ServerLocation和ReplicaLocation
		location = req.ServerLocation
		Chunkid = req.Filename + "-" + strconv.Itoa(int(req.Chunkindex))
	}

	//创建副本
	if willrpc && !replicaAlive {
		location_replica = metas[1].NodeLocation
		Chunkid_replica, replicaAlive = AddChunk(req.Filename, location_replica, req.Chunkindex)
		if !replicaAlive {
			goto funcBegin
		}
	} else if !willrpc { // recover，req中含有ServerLocation和ReplicaLocation
		location_replica = req.ReplicaLocation
		Chunkid_replica = req.Filename + "-" + strconv.Itoa(int(req.Chunkindex))
	}

	for index, meta := range scheduleList {
		if meta.NodeLocation == location {
			meta.ChunkNum++
			scheduleList[index] = meta // 增加该chunkserver中的文件数记录
		}
		if meta.NodeLocation == location_replica {
			meta.ChunkNum++
			scheduleList[index] = meta // 增加该chunkserver中的文件数记录
		}
	}

	chunkslice := fileToChunkMap[req.Filename]
	newchunkmetadata := ChunkMetadata{Chunkid, location, 0}
	newchunkmetadata_replica := ChunkMetadata{Chunkid_replica, location_replica, 0}
	ChunkReplicas := ChunkReplicas{newchunkmetadata, newchunkmetadata_replica}
	chunkslice[req.Chunkindex] = ChunkReplicas
	fileToChunkMap[req.Filename] = chunkslice
	return Chunkid, location, location_replica, 0
}

// ChunkMasterRequset{Filename, 0, false, false}，Filename有用
// 创建新文件会调用的方法，应当给该文件创建一个 chunk (master 直接通信 chunkserver)
// 将该 chunk 加入映射，并返回 Chunkid 和 location。不应该有文件同名
func (this *ChunkMasterHandle) NewFileChunk(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 不是leader
	if !isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("NewFileChunk")
	// 已经被创建了
	if fileToChunkMap[req.Filename] != nil {
		fmt.Println("Master: has already created file")
		res.Errornum = -1
		return nil
	}

	Chunkid, location, location_replica, Errornum := NewFileChunkFunc(req, true)
	log.Println("NewNewChunk filename>" + req.Filename + "/" +
		"Location>" + location + "/" +
		"Location_replica>" + location_replica + "/")

	// 向未抢到锁的master发rpc备份数据
	for i := 8095; i <= 8097; i++ {
		aimMaster := "127.0.0.1:" + strconv.Itoa(i)
		if aimMaster == masterLocation {
			continue // 不用广播给自己
		}

		replicaReq := ChunkMasterRequest{req.Filename, 0, false, false, location, location_replica, "", 0}
		var replicaRes ChunkMasterResponse
		conn, err := rpc.DialHTTP("tcp", aimMaster)
		if err != nil {
			// log.Fatalln("dailing error: ", err)
			fmt.Println("dailing error: ", err)
		}
		defer conn.Close()

		err = conn.Call("ChunkMasterHandle.NewFileChunkReplica", replicaReq, &replicaRes) // 创建新块
		if err != nil {
			// log.Fatalln("ChunkMasterHandle error: ", err)
			fmt.Println("ChunkMasterHandle error: ", err)
		}
	}

	//构造返回请求
	res.Chunkid = Chunkid
	res.Chunklocation = location
	res.Errornum = Errornum
	return nil
}

// NewFileChunk和recovery的时候会调用的函数，返回res中的Chunkid、location和Errornum
func NewFileChunkFunc(req ChunkMasterRequest, willrpc bool) (string, string, string, int) {
funcBegin:
	var Chunkid string = ""
	var Chunkid_replica string = ""
	var ChunkAlive bool = false
	var replicaAlive bool = false
	var location string
	var location_replica string

	var metas []NodeMetadata
	for _, meta := range scheduleList {
		if meta.Alive {
			metas = append(metas, meta)
		}
	}
	sort.Sort(NodeMetadatas(metas)) // 按ChunkNum从小到大排序

	if willrpc && !ChunkAlive {
		// 	for {
		// 		if scheduleList[schedule].Alive {
		// 			break
		// 		}
		// 		schedule = (schedule + 1) % 3
		// 	}
		// 	location = scheduleList[schedule].NodeLocation
		// 	schedule = (schedule + 1) % 3
		location = metas[0].NodeLocation
		Chunkid, ChunkAlive = CreateNewFileChunk(req.Filename, location)
		if !ChunkAlive {
			goto funcBegin
		}
	} else if !willrpc { // recover，req中含有ServerLocation和ReplicaLocation
		location = req.ServerLocation
		Chunkid = req.Filename + "-0"
	}

	//创建副本
	if willrpc && !replicaAlive {
		// 	for {
		// 		if scheduleList[schedule].Alive {
		// 			break
		// 		}
		// 		schedule = (schedule + 1) % 3
		// 	}
		// 	location_replica = scheduleList[schedule].NodeLocation
		// 	schedule = (schedule + 1) % 3
		location_replica = metas[1].NodeLocation
		Chunkid_replica, replicaAlive = CreateNewFileChunk(req.Filename, location_replica)
		if !replicaAlive {
			goto funcBegin
		}
	} else if !willrpc { // recover，req中含有ServerLocation和ReplicaLocation
		location_replica = req.ReplicaLocation
		Chunkid_replica = req.Filename + "-0"
	}

	for index, meta := range scheduleList {
		if meta.NodeLocation == location {
			meta.ChunkNum++
			scheduleList[index] = meta // 增加该chunkserver中的文件数记录
		}
		if meta.NodeLocation == location_replica {
			meta.ChunkNum++
			scheduleList[index] = meta // 增加该chunkserver中的文件数记录
		}
	}

	//加入map中
	newchunkmetadata := ChunkMetadata{Chunkid, location, 0}
	newchunkmetadata_replica := ChunkMetadata{Chunkid_replica, location_replica, 0}
	chunkslice := make(map[uint64]ChunkReplicas)
	ChunkReplicas := ChunkReplicas{newchunkmetadata, newchunkmetadata_replica}
	chunkslice[0] = ChunkReplicas
	fileToChunkMap[req.Filename] = chunkslice
	return Chunkid, location, location_replica, 0
}

// ChunkMasterRequest{Filename, 0, false, false}, Filename有用
// 删除文件会调用的方法，应当清空 chunk (master 直接通信 chunkserver)
func (this *ChunkMasterHandle) DeleteFileAndChunks(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 不是leader
	if !isLeader {
		res.Errornum = -4
		return nil
	}

	log.Println("DeleteChunk filename:" + req.Filename + "/")
	Errornum := DeleteFileAndChunksFunc(req, true)
	// 向未抢到锁的master发rpc备份数据
	for i := 8095; i <= 8097; i++ {
		aimMaster := "127.0.0.1:" + strconv.Itoa(i)
		if aimMaster == masterLocation {
			continue // 不用广播给自己
		}

		replicaReq := req
		var replicaRes ChunkMasterResponse
		conn, err := rpc.DialHTTP("tcp", aimMaster)
		if err != nil {
			// log.Fatalln("dailing error: ", err)
			fmt.Println("dailing error: ", err)
		}
		defer conn.Close()

		err = conn.Call("ChunkMasterHandle.DeleteFileAndChunksReplica", replicaReq, &replicaRes) // 创建新块
		if err != nil {
			// log.Fatalln("ChunkMasterHandle error: ", err)
			fmt.Println("ChunkMasterHandle error: ", err)
		}
	}

	// 构造返回请求
	res.Errornum = Errornum
	return nil
}

// DeleteFileAndChunk和recovery的时候会调用的函数，返回res中的Errornum
func DeleteFileAndChunksFunc(req ChunkMasterRequest, willrpc bool) int {
	chunkslice := fileToChunkMap[req.Filename]
	if willrpc { // 需要发rpc
		wg := sync.WaitGroup{}
		wg.Add(len(chunkslice))
		connection := connect()
		for _, chunkreplicas := range chunkslice {
			chunkmetadata := chunkreplicas.Chunk_1
			chunkmetadata_replica := chunkreplicas.Chunk_2
			location := chunkmetadata.Chunklocation
			Chunkid := chunkmetadata.Chunkid
			location_replica := chunkmetadata_replica.Chunklocation
			Chunkid_replica := chunkmetadata_replica.Chunkid
			go func() {
				lock(connection, "/"+Chunkid)
				DeleteFileChunk(Chunkid, location)
				DeleteFileChunk(Chunkid_replica, location_replica)
				unlock(connection, "/"+Chunkid)
				wg.Done()
			}()
			for index, meta := range scheduleList {
				if meta.NodeLocation == location {
					meta.ChunkNum--
					scheduleList[index] = meta // 减少该chunkserver中的文件数记录
				}
				if meta.NodeLocation == location_replica {
					meta.ChunkNum--
					scheduleList[index] = meta // 减少该chunkserver中的文件数记录
				}
			}
		}
		wg.Wait()
		close(connection)
	} else { // recover时不需要发rpc
		connection := connect()
		for _, chunkreplicas := range chunkslice {
			chunkmetadata := chunkreplicas.Chunk_1
			chunkmetadata_replica := chunkreplicas.Chunk_2
			location := chunkmetadata.Chunklocation
			location_replica := chunkmetadata_replica.Chunklocation

			for index, meta := range scheduleList {
				if meta.NodeLocation == location {
					meta.ChunkNum--
					scheduleList[index] = meta // 减少该chunkserver中的文件数记录
				}
				if meta.NodeLocation == location_replica {
					meta.ChunkNum--
					scheduleList[index] = meta // 减少该chunkserver中的文件数记录
				}
			}

		}
		close(connection)
	}

	delete(fileToChunkMap, req.Filename)
	return 0
}

func (this *ChunkMasterHandle) AcceptHeartbeat(req OperationRequest, res *OperationResponse) error {
	isNew := true // 是不是新加入的chunkserver
	newLocation := req.ServerLocation

	if !isLeader {
		res.Reply = false
		return nil
	}

	for i, meta := range scheduleList {
		if meta.NodeLocation == newLocation {
			scheduleList[i].Alive = true                         // 设为alive
			serverToTimerMap[newLocation].Reset(3 * time.Second) // 重置定时器
			// fmt.Println(scheduleList[i].NodeLocation, " is alive")
			isNew = false
		}
	}

	if isNew { // 是新加入的chunkserver
		serverToTimerMap[newLocation] = time.NewTimer(3 * time.Second) // 加入map
		newNodeMeta := NodeMetadata{newLocation, true, 0}
		scheduleList = append(scheduleList, newNodeMeta) // 加入scheduleList
		fmt.Println(newLocation, " joined")

		go timeup(serverToTimerMap[newLocation], newLocation)
	}

	return nil
}

func timeup(t *time.Timer, serverloaciton string) {
	for {
		<-t.C
		for i, meta := range scheduleList {
			if meta.NodeLocation == serverloaciton {
				scheduleList[i].Alive = false
			}
		}
		fmt.Println(serverloaciton, " is dead")
		t.Reset(3 * time.Second)
	}
}

func recover() int {
	file, err0 := os.OpenFile(path+"/log/master.log", os.O_RDONLY, 0666)
	if err0 != nil { // 说明没有日志文件
		return -1
	}
	defer file.Close()

	rd := bufio.NewReader(file)
	for {
		line, err := rd.ReadString('\n') // 以'\n'为结束符读，即读入一行
		if err != nil || io.EOF == err { // 读到EOF了，结束
			break
		}

		lineValid := line[20:]       // 0-19是时间，从20开始截取有用的信息
		operation := lineValid[0:11] // 操作符是11位的

		switch operation {
		case "AddAddChunk": // 格式 AddAddChunk filename>Filename/Chunkindex>Chunkindex/Location>ServerLocation/Location_replica>ReplicaLocation/
			var greater, slash int // 分别记录>和/的位置
			var tempLineValid string
			greater = strings.Index(lineValid, ">")
			slash = strings.Index(lineValid, "/")
			aimFilename := lineValid[greater+1 : slash]

			tempLineValid = lineValid[slash+1:]
			greater = strings.Index(tempLineValid, ">")
			slash = strings.Index(tempLineValid, "/")
			intChunkindex, _ := strconv.Atoi(string(tempLineValid[greater+1 : slash]))

			tempLineValid = lineValid[slash+1:]
			greater = strings.Index(tempLineValid, ">")
			slash = strings.Index(tempLineValid, "/")
			aimLocation := tempLineValid[greater+1 : slash]

			tempLineValid = lineValid[slash+1:]
			greater = strings.LastIndex(tempLineValid, ">")
			slash = strings.LastIndex(tempLineValid, "/")
			aimLocation_replica := tempLineValid[greater+1 : slash]

			req := ChunkMasterRequest{aimFilename, uint64(intChunkindex), true, false,
				aimLocation, aimLocation_replica, "", 0}
			AddChunkFunc(req, false)

		case "NewNewChunk": // 格式 NewNewChunk filename:Filename/Location:ServerLocation/Location_replica:ReplicaLocation/
			var greater, slash int // 分别记录:和/的位置
			var tempLineValid string
			greater = strings.Index(lineValid, ">")
			slash = strings.Index(lineValid, "/")
			aimFilename := lineValid[greater+1 : slash]

			tempLineValid = lineValid[slash+1:]
			greater = strings.Index(tempLineValid, ">")
			slash = strings.Index(tempLineValid, "/")
			aimLocation := tempLineValid[greater+1 : slash]

			tempLineValid = lineValid[slash+1:]
			greater = strings.LastIndex(tempLineValid, ">")
			slash = strings.LastIndex(tempLineValid, "/")
			aimLocation_replica := tempLineValid[greater+1 : slash]

			req := ChunkMasterRequest{aimFilename, 0, false, false,
				aimLocation, aimLocation_replica, "", 0}
			NewFileChunkFunc(req, false)

		case "DeleteChunk": // 格式 DeleteChunk filename:Filename/
			var colon, slash int // 分别记录:和/的位置
			colon = strings.LastIndex(lineValid, ":")
			slash = strings.LastIndex(lineValid, "/")
			aimFilename := lineValid[colon+1 : slash]

			req := ChunkMasterRequest{aimFilename, 0, false, false, "", "", "", 0}
			DeleteFileAndChunksFunc(req, false)

		case "UpdateVsion": // 格式 UpdateVsion ServerLocation:ServerLocation/Chunkid:Chunkid/Version:Version/
			var colon, slash int // 分别记录:和/的位置
			colon = strings.Index(lineValid, ":")
			slash = strings.Index(lineValid, "/")
			aimServerLocation := lineValid[colon+1 : slash]

			tempLineValid := lineValid[slash+1:]
			colon = strings.Index(tempLineValid, ":")
			slash = strings.Index(tempLineValid, "/")
			aimChunkid := tempLineValid[colon+1 : slash]

			colon = strings.LastIndex(lineValid, ":")
			slash = strings.LastIndex(lineValid, "/")
			intVersion, _ := strconv.Atoi(string(lineValid[colon+1 : slash]))

			req := OperationRequest{aimServerLocation, aimChunkid, uint64(intVersion)}
			UpdateChunkVersionFunc(req)
			// default:
			// 	log.Fatalln("there is an invalid record in log")
		}
	}

	for _, value := range fileToChunkMap {
		fmt.Println(value)
	}

	return 0

}

func main() {
	if len(os.Args) == 1 {
		log.Fatalln("master boot without location")
		return
	}
	masterLocation += string(os.Args[1])

	// 先初始化fileToChunkMap，下面recovery他，如果别的master有就直接copy，没有就通过log恢复
	fileToChunkMap = make(map[string]map[uint64]ChunkReplicas)

	path = string(os.Args[1])
	whether, _ := PathExists(path + "/log")
	if !whether {
		err := os.MkdirAll(path+"/log", os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	recover()
	logFile, logErr := os.OpenFile(path+"/log/master.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if logErr != nil {
		log.Fatalln(logErr)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	connection := connect()
	defer close(connection)
	chRetry := make(chan bool)
	chGetLock := make(chan bool)
	isFirst := true

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
	becomeLeader:
		go lockmaster(connection, string(os.Args[1]), chRetry, chGetLock)
		isLeader = <-chGetLock
		if isFirst { // 只有第一次的时候才会wg.Done，后面再尝试成为leader则不会
			wg.Done()
			isFirst = false
		}

		if !isLeader { // 没有成为leader，一直监听，等待leader死掉
			isLeader = false

			available := <-chRetry // 等待watchMaster把值传入channel
			if available {
				goto becomeLeader
			}
		} else {
			serverToTimerMap = make(map[string]*time.Timer)

			isLeader = true //把全局变量设为true，代表是leader
			// 开始接收心跳
			serverToTimerMap[nodeLocation_1] = time.NewTimer(3 * time.Second)
			serverToTimerMap[nodeLocation_2] = time.NewTimer(3 * time.Second)
			serverToTimerMap[nodeLocation_3] = time.NewTimer(3 * time.Second)
			go timeup(serverToTimerMap[nodeLocation_1], nodeLocation_1)
			go timeup(serverToTimerMap[nodeLocation_2], nodeLocation_2)
			go timeup(serverToTimerMap[nodeLocation_3], nodeLocation_3)
			fmt.Println("become leader")
		}
	}()

	// rpc.Register(new(ReplicaMasterHandle)) // 注册rpc服务
	// rpc.HandleHTTP()                       // 采用http协议作为rpc载体

	// fmt.Println("replica master start!")
	// lis, err := net.Listen("tcp", masterLocation)
	// if err != nil {
	// 	log.Fatalln("fatal error: ", err)
	// }
	// fmt.Println("replica master start connection!")
	// go http.Serve(lis, nil)

	// 后面要加上从 log 文件读取内容来填充 fileToChunkMap
	wg.Wait()
	rpc.Register(new(ChunkMasterHandle)) // 注册rpc服务
	rpc.HandleHTTP()                     // 采用http协议作为rpc载体

	if isLeader {
		fmt.Println("leader master start!")
	} else {
		fmt.Println("replica master start!")
	}

	lis, err := net.Listen("tcp", masterLocation)
	if err != nil {
		// log.Fatalln("fatal error: ", err)
		fmt.Println("fatal error: ", err)
	}

	if isLeader {
		fmt.Println("leader master start connection!")
	} else {
		fmt.Println("replica master start connection!")
	}

	http.Serve(lis, nil)
}
