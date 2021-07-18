package main

import (
	"fmt"
	"log"
	"strconv"
)

// ChunkMasterRequest{Filename, Chunkindex, true, false}，Filename和Chunkindex有用
// 增加chunk方法
func (this *ChunkMasterHandle) AddChunkReplica(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 是leader
	if isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("Replica AddChunk")
	//没有该chunk
	if fileToChunkMap[req.Filename] == nil {
		res.Errornum = -2
		return nil
	}

	Chunkindex := strconv.Itoa(int(req.Chunkindex))
	log.Println("AddAddChunk filename>" + req.Filename + "/Chunkindex>" + Chunkindex + "/" +
		"Location>" + req.ServerLocation + "/" +
		"Location_replica>" + req.ReplicaLocation + "/")

	Chunkid, location, _, Errornum := AddChunkFunc(req, false)
	res.Chunkid = Chunkid
	res.Chunklocation = location
	res.Errornum = Errornum
	return nil
}

// ChunkMasterRequset{Filename, 0, false, false}，Filename有用
// 创建新文件会调用的方法，应当给该文件创建一个 chunk (master 直接通信 chunkserver)
// 将该 chunk 加入映射，并返回 Chunkid 和 location。不应该有文件同名
func (this *ChunkMasterHandle) NewFileChunkReplica(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 是leader
	if isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("Replica NewFileChunk")
	// 已经被创建了
	if fileToChunkMap[req.Filename] != nil {
		fmt.Println("Master: has already created file")
		res.Errornum = -1
		return nil
	}

	log.Println("NewNewChunk filename>" + req.Filename + "/" +
		"Location>" + req.ServerLocation + "/" +
		"Location_replica>" + req.ReplicaLocation + "/")
	Chunkid, location, _, Errornum := NewFileChunkFunc(req, false)

	//构造返回请求
	res.Chunkid = Chunkid
	res.Chunklocation = location
	res.Errornum = Errornum
	return nil
}

// ChunkMasterRequest{Filename, 0, false, false}, Filename有用
// 删除文件会调用的方法，应当清空 chunk (master 直接通信 chunkserver)
func (this *ChunkMasterHandle) DeleteFileAndChunksReplica(req ChunkMasterRequest, res *ChunkMasterResponse) error {
	// 是leader
	if isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("Replica DeleteFileAndChunks")
	log.Println("DeleteChunk filename:" + req.Filename + "/")
	Errornum := DeleteFileAndChunksFunc(req, false)

	// 构造返回请求
	res.Errornum = Errornum
	return nil
}

func (this *ChunkMasterHandle) UpdateChunkVersionReplica(req OperationRequest, res *OperationResponse) error {
	// 是leader
	if isLeader {
		res.Errornum = -4
		return nil
	}

	fmt.Println("Replica UpdateChunkVersion")
	Version := strconv.Itoa(int(req.Version))
	log.Println("UpdateVsion ServerLocation:" + req.ServerLocation +
		"/Chunkid:" + req.Chunkid +
		"/Version:" + Version +
		"/")
	Reply := UpdateChunkVersionFunc(req)
	res.Reply = Reply
	return nil
}
