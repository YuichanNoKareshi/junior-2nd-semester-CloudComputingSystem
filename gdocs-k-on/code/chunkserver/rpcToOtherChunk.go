package main

import (
	"fmt"
	"log"
	// "net"
	// "net/http"
	"net/rpc"
	// "os"
	"strconv"
)

type BroadcastRequest struct {
	Chunkid   string
	Version   uint64
	Bytebegin uint64
	Byteend   uint64
	Data      string //写的时候要给的数据
}

type BroadcastResponse struct {
	Reply bool
}

// originReq是收到rpc的chunkserver直接把转发的req
func updateReplicaVersion(originReq ChunkRequest, Version uint64, ServerLocation string) error {
	fmt.Println("updateVersion in other chunkserver")
	updateNum := 0 // 记录广播后更新的chunk数，是1则正常，是0或2则不正常

	allServerLocation := originReq.AllServerLocation
	fmt.Println(allServerLocation)
	for _, aimChunkServer := range allServerLocation {
		if aimChunkServer == ServerLocation {
			continue // 不用广播给自己
		}

		conn, err := rpc.DialHTTP("tcp", aimChunkServer)
		if err != nil { // 说明有chunkserver死掉了
			// log.Fatalln("dailing error: ", err)
			continue
		}
		defer conn.Close()

		req := BroadcastRequest{originReq.Chunkid, Version, originReq.Bytebegin, originReq.Byteend, originReq.Data}
		var res BroadcastResponse

		err = conn.Call("ChunkHandle.UpdateOwnChunkVersion", req, &res)
		if err != nil {
			fmt.Println("BroadcastHandle error: ", err)
		}

		if res.Reply {
			updateNum++
		}
	}

	if updateNum != 1 {
		fmt.Println("BroadcastHandle updated ", updateNum, " chunk")
	}
	return nil
}

/* req
Chunkid        string
Version        uint64
Bytebegin      uint64
Byteend        uint64
Data           string */
func (this *ChunkHandle) UpdateOwnChunkVersion(req BroadcastRequest, res *BroadcastResponse) error {
	fmt.Println("update own chunkVersion")

	fmt.Println("aim chunkid: ", req.Chunkid)
	for key, _ := range nameToChunkMetadataMap {
		fmt.Println("chunkid: ", key)
	}
	chunkmetadata, isExist := nameToChunkMetadataMap[req.Chunkid] // 找到该chunkserver中的replica chunk
	if !isExist {                                                 // 该chunkserver中没有对应的replica chunk
		res.Reply = false
		return nil
	}

	var tempStrings []string
	originReq := ChunkRequest{req.Chunkid, "", req.Bytebegin, req.Byteend, req.Data, tempStrings, 0}
	WriteChunk(originReq) // 向replica chunk中更新内容(进行同样的写操作)

	// 下面开始更新version
	chunkmetadata.Version = req.Version
	nameToChunkMetadataMap[req.Chunkid] = chunkmetadata
	updateMasterVersion(req.Chunkid, chunkmetadata.Version, serverLocation) // 更新master中记录的该每个chunk的version
	log.Println("Version:" + strconv.Itoa(int(chunkmetadata.Version)) + "-" + req.Chunkid + "/")
	res.Reply = true

	return nil
}
