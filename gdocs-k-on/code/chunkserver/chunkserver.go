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
	"strconv"
	"strings"
	"time"
)

var serverLocation string = "127.0.0.1:"

const chunk_size = 1024 * 1024

var path string

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

// chunk的名字对应的元数据的map，key是chunkid，value是chunk(metadata)
var nameToChunkMetadataMap map[string]Chunk

// chunk 元数据结构
// 只保存最高 Version 的块的元数据
type Chunk struct {
	Chunkid string
	Version uint64
}

// chunk 管理结构体
type ChunkHandle struct {
}

// chunk 管理请求结构体
type ChunkRequest struct {
	Chunkid           string
	Filename          string //只有master会用到这个参数 创建和删除时会传过来
	Bytebegin         uint64
	Byteend           uint64
	Data              string //写时要给的数据
	AllServerLocation []string
	Index             uint64
}

// chunk 管理响应结构体
type ChunkResponse struct {
	Data     string //读时要返回的数据
	Chunkid  string //只有新块创建的时候会用到
	Errornum int    // 0 正常
}

// // 更新某chunk后广播用的结构体
// type BroadcastRequest struct {
// 	Chunkid   string
// 	Version   uint64
// 	Bytebegin uint64
// 	Byteend   uint64
// 	Data      string //写的时候要给的数据
// }

// type BroadcastResponse struct {
// 	Reply bool
// }

//Version 暂时还不知道咋用 可能应该写在文件里
//创建新块会调用的方法，默认名是Filename-0
func (this *ChunkHandle) Create(req ChunkRequest, res *ChunkResponse) error {
	Chunkid := CreateChunk(req)
	res.Chunkid = Chunkid
	res.Errornum = 0

	log.Println("Version:0-" + Chunkid + "/") // 只有调rpc时才写log
	return nil
}

// rpc调用Create和recovery时会调用该函数
func CreateChunk(req ChunkRequest) string { // 返回Chunkid
	Chunkid := req.Filename + "-0"
	log.Println("Createe Chunkid:" + Chunkid + "/")
	f, err := os.Create(path + "/" + Chunkid)
	//f, err := os.OpenFile(path+"/"+Chunkid,  os.O_CREATE | os.O_WRONLY | os.O_APPEND,os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	//os.Chmod(path+"/"+Chunkid,0777)

	_, err = f.Seek(chunk_size-1, 0)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = f.Write([]byte{0})
	if err != nil {
		log.Fatalln(err)
	}

	chunkmetadata := Chunk{Chunkid, 0} // Version 为0
	nameToChunkMetadataMap[Chunkid] = chunkmetadata
	return Chunkid
}

//往某一个文件add块
func (this *ChunkHandle) Add(req ChunkRequest, res *ChunkResponse) error {
	Chunkid := AddChunk(req)
	res.Chunkid = Chunkid
	res.Errornum = 0

	log.Println("Version:0-" + Chunkid)
	return nil
}

// rpc调用Add和recovery时会调用该函数
func AddChunk(req ChunkRequest) string { // 返回Chunkid
	strindex := strconv.Itoa(int(req.Index))
	Chunkid := req.Filename + "-" + strindex
	log.Println("Createe Chunkid:" + Chunkid + "/")
	f, err := os.Create(path + "/" + Chunkid)
	//f, err := os.OpenFile(path+"/"+Chunkid,  os.O_CREATE | os.O_WRONLY | os.O_APPEND,os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	//os.Chmod(path+"/"+Chunkid,0777)

	_, err = f.Seek(chunk_size-1, 0)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = f.Write([]byte{0})
	if err != nil {
		log.Fatalln(err)
	}

	chunkmetadata := Chunk{Chunkid, 0} //Version 为0
	nameToChunkMetadataMap[Chunkid] = chunkmetadata
	return Chunkid
}

//删除块会调用的方法
func (this *ChunkHandle) Delete(req ChunkRequest, res *ChunkResponse) error {
	DeleteChunk(req)
	log.Println("Removee Chunkid:" + req.Chunkid + "/")
	res.Errornum = 0
	return nil
}

// rpc调用Delete和recovery时会调用该函数
func DeleteChunk(req ChunkRequest) {
	err := os.Remove(path + "/" + req.Chunkid)
	if err != nil {
		log.Fatalln(err)
	}
	delete(nameToChunkMetadataMap, req.Chunkid)
}

func (this *ChunkHandle) Read(req ChunkRequest, res *ChunkResponse) error {
	Chunkid := req.Chunkid
	begin := req.Bytebegin
	end := req.Byteend
	fmt.Print("Read begin: ")
	fmt.Println(begin)
	fmt.Print("Read end: ")
	fmt.Println(end)
	file, err := os.Open(path + "/" + Chunkid)
	//file, err := os.OpenFile(path+"/"+Chunkid,   os.O_WRONLY,0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	file.Seek(int64(begin), 0)
	buf := make([]byte, end-begin)
	_, err2 := file.Read(buf)
	if err2 == io.EOF { // io.EOF表示文件末尾
		fmt.Println("文件读取结束")
	}
	res.Data = string(buf)
	fmt.Println(buf)
	fmt.Println("chunkserver data response: " + res.Data)
	res.Errornum = 0
	return nil
}

// 来自gfs-api
/* req
Chunkid   string
Filename  string
Bytebegin uint64
Byteend   uint64
Data      string */
func (this *ChunkHandle) Write(req ChunkRequest, res *ChunkResponse) error {
	Chunkid := req.Chunkid
	res.Errornum = WriteChunk(req)

	// 下面开始更新version
	chunkmetadata := nameToChunkMetadataMap[Chunkid]
	chunkmetadata.Version = chunkmetadata.Version + 1
	nameToChunkMetadataMap[Chunkid] = chunkmetadata
	updateMasterVersion(Chunkid, chunkmetadata.Version, serverLocation) // 更新master中记录的该chunk的version
	updateReplicaVersion(req, chunkmetadata.Version, serverLocation)    // 更新该chunk的replica使其与自己一致
	log.Println("Version:" + strconv.Itoa(int(chunkmetadata.Version)) + "-" + Chunkid + "/")
	return nil
}

// 用来根据req写chunk，在Write和UpdateOwnChunkVersion中会被调用，不涉及Version的更新操作
/* req
Chunkid   string
Filename  string
Bytebegin uint64
Byteend   uint64
Data      string */
func WriteChunk(req ChunkRequest) int {
	Chunkid := req.Chunkid
	begin := req.Bytebegin // 从begin开始写
	end := req.Byteend     // 写到end
	log.Println("Writeee Chunkid:" + Chunkid +
		"/Begin:" + strconv.Itoa(int(begin)) +
		"/End:" + strconv.Itoa(int(end)) +
		"/Data:" + req.Data +
		"/") // 有效信息都是以:开头，以/结尾
	if end >= chunk_size {
		log.Fatalln("Write bytes exceed")
	}
	//file, err := os.Open(path+"/"+Chunkid)
	file, err := os.OpenFile(path+"/"+Chunkid, os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	file.Seek(int64(begin), 0)
	// _,err2 := file.WriteString(req.Data)
	_, err2 := io.WriteString(file, req.Data)
	if err2 != nil { // io.EOF表示文件末尾
		log.Fatalln(err2)
	}

	return 0 // 作为res的Errornum
}

func heartbeat() {
	for {
		time.Sleep(1 * time.Second)
		sendHeartbeat()
	}
}

func recover() int {
	file, err0 := os.OpenFile(path+"/log/chunkserver.log", os.O_RDONLY, 0666)
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

		lineValid := line[20:]      // 0-19是时间，从20开始截取有用的信息
		operation := lineValid[0:7] // 操作符是7位的

		switch operation {
		case "Createe": // 格式 Createe Chunkid:xxx-y
			slash := strings.Index(lineValid, "/")
			dash := strings.LastIndex(lineValid, "-")
			aimFilename := lineValid[16:dash]
			intChunknum, _ := strconv.Atoi(string(lineValid[dash+1 : slash]))
			var tempStrings []string
			req := ChunkRequest{"", aimFilename, 0, 0, "", tempStrings, uint64(intChunknum)}
			if intChunknum == 0 { // ChunkNum是0就调用create，不是0就调用add
				CreateChunk(req)
			} else {
				AddChunk(req)
			}

		case "Writeee": // 格式 Writeee Chunkid:xxx-y/Begin:a/End:b/Data:efg/
			var colon, slash int // 分别记录:和/的位置
			var lineTemp string  // 临时字符串
			lineTemp = lineValid
			colon = strings.Index(lineTemp, ":")
			slash = strings.Index(lineTemp, "/")
			aimChunkid := lineValid[colon+1 : slash]

			lineTemp = lineTemp[slash+1:]
			colon = strings.Index(lineTemp, ":")
			slash = strings.Index(lineTemp, "/")
			intBegin, _ := strconv.Atoi(string(lineTemp[colon+1 : slash]))
			aimBegin := uint64(intBegin)

			lineTemp = lineTemp[slash+1:]
			colon = strings.Index(lineTemp, ":")
			slash = strings.Index(lineTemp, "/")
			intEnd, _ := strconv.Atoi(string(lineTemp[colon+1 : slash]))
			aimEnd := uint64(intEnd)

			lineTemp = lineTemp[slash+1:]
			colon = strings.Index(lineTemp, ":")
			slash = strings.LastIndex(lineTemp, "/")
			aimData := lineTemp[colon+1 : slash]

			var tempStrings []string
			req := ChunkRequest{aimChunkid, "", aimBegin, aimEnd, aimData, tempStrings, 0}
			WriteChunk(req)

		case "Version": // 格式 Version:z-xxx-y (version + chunkid)
			intVersion, _ := strconv.Atoi(string(lineValid[8]))
			aimVersion := uint64(intVersion)
			slash := strings.Index(lineValid, "/")
			aimChunkid := lineValid[10:slash]

			//下面更新Chunkid的Version
			chunkmetadata := nameToChunkMetadataMap[aimChunkid]
			chunkmetadata.Version = aimVersion
			nameToChunkMetadataMap[aimChunkid] = chunkmetadata

		case "Removee": // 格式 Removee Chunkid:xxx-y/
			colon := strings.Index(lineValid, ":")
			slash := strings.Index(lineValid, "/")
			aimChunkid := lineValid[colon+1 : slash]
			var tempStrings []string
			req := ChunkRequest{aimChunkid, "", 0, 0, "", tempStrings, 0}
			DeleteChunk(req)
			// default:
			// 	log.Fatalln("there is an invalid record in log")
		}
	}

	return 0
}

func main() {
	nameToChunkMetadataMap = make(map[string]Chunk)
	// 后面要加上从 log 文件读取内容来填充 nameToChunkMetadataMap
	// log 日志
	if len(os.Args) == 1 {
		log.Fatalln("chunkserver boot without location")
		return
	}
	serverLocation += string(os.Args[1])
	path = string(os.Args[1])
	whether, _ := PathExists(path + "/log")
	if !whether {
		fmt.Println("mkdir")
		err := os.MkdirAll(path+"/log", os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	recover()
	// if isLogExist == -1 { // 没有log文件
	logFile, logErr := os.OpenFile(path+"/log/chunkserver.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if logErr != nil {
		log.Fatalln(logErr)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	// }

	rpc.Register(new(ChunkHandle)) // 注册rpc服务
	rpc.HandleHTTP()               // 采用http协议作为rpc载体

	fmt.Println("chunkserver start!")
	lis, err := net.Listen("tcp", serverLocation)
	if err != nil {
		// log.Fatalln("fatal error: ", err)
		fmt.Println("fatal error: ", err)
	}

	fmt.Println("chunkserver start connection on " + serverLocation)
	go heartbeat() // 开始发送心跳

	http.Serve(lis, nil)
}
