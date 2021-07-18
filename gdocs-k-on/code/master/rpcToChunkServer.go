package main

import (
	"fmt"
	"log"
	// "net"
	// "net/http"
	"net/rpc"
	"strconv"
	"strings"
)

// chunk 管理请求结构体
type ChunkRequest struct {
	Chunkid   string
	Filename  string //只有master会用到这个参数 创建和删除时会传过来
	Bytebegin uint64
	Byteend   uint64
	Index     uint64
}

// chunk 管理响应结构体
type ChunkResponse struct {
	Data     string
	Chunkid  string //只有新块创建的时候会用到
	Errornum int    // 0 正常
}

// from master to chunkserver，如果chunkserver死了会返回false，跳过该chunkserver，在下一个chunkserver里加
func CreateNewFileChunk(Filename string, location string) (string, bool) {
	conn, err := rpc.DialHTTP("tcp", location)
	if err != nil { // 如果到了这里就说明这个chunkserver死掉了
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailinig error: ", err)
		return "", false
	}
	defer conn.Close()

	req := ChunkRequest{"", Filename, 0, 0, 0}
	var res ChunkResponse

	err = conn.Call("ChunkHandle.Create", req, &res) // 创建新块
	if err != nil {
		// log.Fatalln("ChunkHandle error: ", err)
		fmt.Println("ChunkHandle error: ", err)
	}
	return res.Chunkid, true
}

func AddChunk(Filename string, location string, index uint64) (string, bool) {
	conn, err := rpc.DialHTTP("tcp", location)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailinig error: ", err)
		return "", false
	}
	defer conn.Close()

	fmt.Println("Add chunk:", index)
	req := ChunkRequest{"", Filename, 0, 0, index}
	var res ChunkResponse

	fmt.Println(req)
	err = conn.Call("ChunkHandle.Add", req, &res) // 增加新块
	if err != nil {
		// log.Fatalln("ChunkHandle error: ", err)
		fmt.Println("ChunkHandle error: ", err)
	}
	return res.Chunkid, true
}

func DeleteFileChunk(Chunkid string, location string) (string, bool) {
	conn, err := rpc.DialHTTP("tcp", location)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailinig error: ", err)
		return "", false
	}
	defer conn.Close()

	req := ChunkRequest{Chunkid, "", 0, 0, 0}
	var res ChunkResponse

	err = conn.Call("ChunkHandle.Delete", req, &res) // 删除块
	if err != nil {
		// log.Fatalln("ChunkHandle error: ", err)
		fmt.Println("ChunkHandle error: ", err)
	}
	return res.Chunkid, true
}

//from chunkser to master

// chunkserver to master 操作 管理请求结构体
type OperationRequest struct {
	ServerLocation string
	// 告诉 master 该chunk已经更新完毕的相关元数据
	Chunkid string
	Version uint64
}

// chunkserver to master 操作 管理响应结构体
type OperationResponse struct {
	Reply    bool
	Errornum int
}

// OperationRequest{ServerLocation, Chunkid, Version}，三个都有用
func (this *ChunkMasterHandle) UpdateChunkVersion(req OperationRequest, res *OperationResponse) error {
	// 不是leader
	if !isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("master update version")

	Version := strconv.Itoa(int(req.Version))
	log.Println("UpdateVsion ServerLocation:" + req.ServerLocation +
		"/Chunkid:" + req.Chunkid +
		"/Version:" + Version +
		"/")
	Reply := UpdateChunkVersionFunc(req)

	// 向未抢到锁的master发rpc备份数据
	for i := 8095; i <= 8097; i++ {
		aimMaster := "127.0.0.1:" + strconv.Itoa(i)
		if aimMaster == masterLocation {
			continue // 不用广播给自己
		}

		replicaReq := req
		var replicaRes OperationResponse
		conn, err := rpc.DialHTTP("tcp", aimMaster)
		if err != nil {
			// log.Fatalln("dailing error: ", err)
			fmt.Println("dailing error: ", err)
		}
		defer conn.Close()

		err = conn.Call("ChunkMasterHandle.UpdateChunkVersionReplica", replicaReq, &replicaRes) // 创建新块
		if err != nil {
			// log.Fatalln("ChunkMasterHandle error: ", err)
			fmt.Println("ChunkMasterHandle error: ", err)
		}
	}

	res.Reply = Reply
	return nil
}

// UpdateChunkVersion和recovery会调用的函数，返回res中的Reply
func UpdateChunkVersionFunc(req OperationRequest) bool {
	serverlocation := req.ServerLocation
	chunkid := req.Chunkid
	version := req.Version

	dash := strings.LastIndex(chunkid, "-")
	id, _ := strconv.Atoi(chunkid[(dash + 1):]) // 是该文件的第几个chunk
	filename := chunkid[:dash]

	chunkreplicas := fileToChunkMap[filename][uint64(id)] // 找到该文件该chunk的备份

	chunk1 := chunkreplicas.Chunk_1
	chunk2 := chunkreplicas.Chunk_2
	if chunk1.Chunklocation == serverlocation {
		// 更新master中存的该chunk的version
		chunk1.Version = version
		chunkreplicas.Chunk_1 = chunk1
	} else if chunk2.Chunklocation == serverlocation {
		chunk2.Version = version
		chunkreplicas.Chunk_2 = chunk2
	} else {
		log.Println("error serverloaciton")
	}

	chunkslice := make(map[uint64]ChunkReplicas)
	// chunkslice[uint64(id)] = chunkreplicas
	// chunkslice := fileToChunkMap[filename]
	for chunkid := range fileToChunkMap[filename] {
		chunkslice[chunkid] = fileToChunkMap[filename][chunkid]
	}
	chunkslice[uint64(id)] = chunkreplicas
	fileToChunkMap[filename] = chunkslice
	// fileToChunkMap[filename][uint64(id)] = chunkreplicas

	return true
}

func (this *ChunkMasterHandle) Test(req OperationRequest, res *OperationResponse) error {

	fmt.Println("test")
	res.Reply = true

	return nil
}
