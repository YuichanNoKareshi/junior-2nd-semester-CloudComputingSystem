package main

import (
	"fmt"
	// "log"
	// "net"
	// "net/http"
	"net/rpc"
	// "os"
	"time"
)

var masterIP string = "127.0.0.1:"

var masterLocation string

// // chunkmaster 管理请求结构体
// type ChunkMasterRequest struct {
// 	ServerLocation string
// 	// 告诉 master 该chunk已经更新完毕的相关元数据
// 	Chunkid string
// 	Version uint64
// }

// // chunkmaster 管理响应结构体
// type ChunkMasterResponse struct {
// 	Reply bool
// }

// chunkserver to master 操作 管理请求结构体
type OperationRequest struct {
	ServerLocation string
	// 告诉 master 该chunk已经更新完毕的相关元数据
	Chunkid string
	Version uint64
}

// chunkserver to master 操作 管理响应结构体
type OperationResponse struct {
	Reply bool
}

func getMasterLocation() {
	connection := connect()
	port, portErr := get(connection, "/master")
	for portErr == false {
		port, portErr = get(connection, "/master") // 获得当前master的port
	}
	close(connection)

	masterLocation = masterIP + port
}

func updateMasterVersion(Chunkid string, Version uint64, ServerLocation string) error {
	fmt.Println("updateVersion in master")
	goto functionBegin
	// dialLeader:
	// 	fmt.Println("updateVersion in master")
	// 	connection := connect()
	// 	port, portErr := get(connection, "/master")
	// 	for portErr == false {
	// 		port, portErr = get(connection, "/master") // 获得当前master的port
	// 	}
	// 	close(connection)

	// 	conn, err := rpc.DialHTTP("tcp", masterLocation+port)
	// 	if err != nil {
	// 		// log.Fatalln("dailing error: ", err)
	// 		fmt.Println("dailing error: ", err)
	// 		// 这时候是leader死掉了，正在等待新的master，要从头开始
	// 		goto dialLeader
	// 	}
	// 	defer conn.Close()
dialLeader:
	getMasterLocation()

functionBegin:
	conn, err := rpc.DialHTTP("tcp", masterLocation)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailing error: ", err)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer conn.Close()

	req := OperationRequest{ServerLocation, Chunkid, Version}
	var res OperationResponse

	err = conn.Call("ChunkMasterHandle.UpdateChunkVersion", req, &res)
	if err != nil {
		fmt.Println("OperationHandle error: ", err)
	}
	fmt.Println(res.Reply)
	if !res.Reply {
		fmt.Println("OperationHandle not reply")
	}
	return nil
}

func sendHeartbeat() {
	goto functionBegin
dialLeader:
	getMasterLocation()

functionBegin:
	conn, err := rpc.DialHTTP("tcp", masterLocation)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailing error: ", err)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer conn.Close()

	req := OperationRequest{serverLocation, "", 0}
	res := OperationResponse{true}

	err = conn.Call("ChunkMasterHandle.AcceptHeartbeat", req, &res)
	if err != nil {
		fmt.Println("Heartbeat error: ", err)
	}
	if res.Reply == false {
		time.Sleep(3 * time.Second)
	}
}
