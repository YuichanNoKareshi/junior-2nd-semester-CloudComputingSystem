package main

import (
	"fmt"
	// "log"
	"net/rpc"
)

var masterLocation string = "127.0.0.1:"

// chunkmaster 管理请求结构体
type ChunkMasterRequest struct {
	Filename   string
	Chunkindex uint64
	Write      bool
	Read       bool
}

// chunkmaster 管理响应结构体
type ChunkMasterResponse struct {
	Chunkid           string
	Chunklocation     string
	Errornum          int
	AllServerLocation []string
}

// chunk 管理请求结构体
type ChunkRequest struct {
	Chunkid           string
	Filename          string //只有master会用到这个参数 创建和删除时会传过来
	Bytebegin         uint64
	Byteend           uint64
	Data              string //写的时候要给的数据
	AllServerLocation []string
}

// chunk 管理响应结构体
type ChunkResponse struct {
	Data     string
	Chunkid  string //只有新块创建的时候会用到
	Errornum int    // 0 正常
}

// default chunk is 1mb
const chunk_size = 1024 * 1024

// create a file
// a file is default one chunk
// 不需要加锁
func create(Filename string) bool {
dialLeader:
	fmt.Println("create")
	connection := connect()
	port, portErr := get(connection, "/master")
	for portErr == false {
		port, portErr = get(connection, "/master") // 获得当前master的port
	}
	close(connection)

	conn, err := rpc.DialHTTP("tcp", masterLocation+port)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailing error: ", err)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer conn.Close()

	req := ChunkMasterRequest{Filename, 0, false, false}
	var res ChunkMasterResponse

	err = conn.Call("ChunkMasterHandle.NewFileChunk", req, &res) // 创建新块
	if err != nil {
		// log.Fatalln("ChunkMasterHandle error: ", err)
		fmt.Println("ChunkMasterHandle error: ", err)
	}
	if res.Errornum == -1 {
		fmt.Println("return false")
		return false
	}
	if res.Errornum == -4 {
		fmt.Println("conntact not leader")
		return false
	}
	return true
}

// write a file
// may add chunk
// 在此对一个chunk加锁
func write(Filename string, offset uint64, Data string) bool {
	fmt.Println("write")
	chunk_index := offset / chunk_size
	chunk_offset := offset % chunk_size
	fmt.Println("offset: ", offset)
	fmt.Println("chunk_index: ", chunk_index)
	fmt.Println("chunk_offset: ", chunk_offset)
	fmt.Println("data len:", uint64(len(Data)))
	if chunk_offset+uint64(len(Data)) >= chunk_size { // 跨chunk写
		var tempdata string
		leftdata := Data          // 记录Data中还未被写入chunk的、剩下的数据
		var lastoffset uint64 = 0 // 记录上一次Data中被写入chunk的字符的起始位置
		connection := connect()
		for chunk_offset+uint64(len(leftdata)) >= chunk_size {
			newsize := chunk_size - chunk_offset             // 该chunk还剩这么多需要写
			tempdata = Data[lastoffset : lastoffset+newsize] // 会取从lastoffset到lastoffset+newsize共newsize个字符
			leftdata = Data[lastoffset+newsize : len(Data)]
			//get chunk location (rpc to master)
			fmt.Println("newsize", newsize)
			fmt.Println(chunk_index)
			fmt.Println(uint64(len(tempdata)))
			location, Chunkid, errornum, allServerLocation := rpcToMasterGetChunkLocation(Filename, chunk_index, true, false)
			if errornum == -2 {
				fmt.Println("write: file does not exist")
				return false
			}
			if errornum == -3 {
				fmt.Println("add chunk:", chunk_index)
				location, Chunkid, errornum = rpcToMasterAddChunk(Filename, chunk_index)
				if errornum == -2 {
					fmt.Println("write: add chunk's file does not exist")
					return false
				}
			}

			lock(connection, "/"+Chunkid)
			fmt.Println("write lock")
			//write chunk (rpc to chunkserver)
			_ = rpcToNodeWriteChunk(location, Chunkid, chunk_offset, tempdata, allServerLocation)
			unlock(connection, "/"+Chunkid)
			fmt.Println("write unlock")

			chunk_offset = (chunk_offset + newsize) % chunk_size
			chunk_index++
			lastoffset += newsize
		}
		if uint64(len(leftdata)) > 0 {
			location, Chunkid, errornum, allServerLocation := rpcToMasterGetChunkLocation(Filename, chunk_index, true, false)
			fmt.Println(allServerLocation)
			if errornum == -2 {
				fmt.Println("write: file does not exist")
				return false
			}
			if errornum == -3 {
				fmt.Println("add chunk:", chunk_index)
				location, Chunkid, errornum = rpcToMasterAddChunk(Filename, chunk_index)
				if errornum == -2 {
					fmt.Println("write: add chunk's file does not exist")
					return false
				}
			}

			lock(connection, "/"+Chunkid)
			fmt.Println("write lock")
			//write chunk (rpc to chunkserver)
			_ = rpcToNodeWriteChunk(location, Chunkid, chunk_offset, leftdata, allServerLocation)
			unlock(connection, "/"+Chunkid)
			fmt.Println("write unlock")
		}
		close(connection)
	} else {
		//get chunk location (rpc to master), location就是在哪个node
		location, Chunkid, errornum, allServerLocation := rpcToMasterGetChunkLocation(Filename, chunk_index, true, false)

		if errornum == -2 {
			fmt.Println("write: file does not exist")
			return false
		}

		//write chunk (rpc to chunkserver)
		connection := connect()
		lock(connection, "/"+Chunkid)
		fmt.Println("write lock")
		_ = rpcToNodeWriteChunk(location, Chunkid, chunk_offset, Data, allServerLocation)
		unlock(connection, "/"+Chunkid)
		fmt.Println("write unlock")
		close(connection)
	}
	return true
}

// read a file
// 在此对一个chunk加锁
func read(Filename string, offset uint64, size uint64) string {
	fmt.Println("read")
	chunk_index := offset / chunk_size
	chunk_offset := offset % chunk_size
	fmt.Println(chunk_index)
	fmt.Println(chunk_offset)
	if size == 0 {
		return ""
	}
	Data := ""
	newsize := size
	lastsize := size
	if chunk_offset+lastsize >= chunk_size {
		connection := connect()
		for chunk_offset+lastsize >= chunk_size {
			newsize = chunk_size - chunk_offset
			lastsize = size - newsize
			//get chunk location (rpc to master)
			location, Chunkid, errornum, _ := rpcToMasterGetChunkLocation(Filename, chunk_index, false, true)
			if errornum == -2 {
				fmt.Println("read: file does not exist")
				return ""
			}
			if errornum == -3 {
				fmt.Println("read: chunk does not exist")
				break
			}

			lock(connection, "/"+Chunkid)
			fmt.Println("read lock")
			//read chunk (rpc to chunkserver)
			tempdata, _ := rpcToNodeReadChunk(location, Chunkid, chunk_offset, newsize)
			unlock(connection, "/"+Chunkid)
			fmt.Println("read unlock")

			chunk_offset = (chunk_offset + newsize) % chunk_size
			chunk_index++
			Data += tempdata
		}
		if lastsize > 0 {
			location, Chunkid, errornum, _ := rpcToMasterGetChunkLocation(Filename, chunk_index, false, true)
			if errornum == -2 {
				fmt.Println("read: file does not exist")
				return ""
			}
			if errornum == -3 {
				fmt.Println("read: chunk does not exist")
				return Data
			}

			lock(connection, "/"+Chunkid)
			fmt.Println("read lock")
			//read chunk (rpc to chunkserver)
			tempdata, _ := rpcToNodeReadChunk(location, Chunkid, chunk_offset, lastsize)
			unlock(connection, "/"+Chunkid)
			fmt.Println("read unlock")
			Data += tempdata
		}
		close(connection)
	} else {
		//get chunk location (rpc to master)
		location, Chunkid, errornum, _ := rpcToMasterGetChunkLocation(Filename, chunk_index, false, true)

		if errornum == -2 {
			fmt.Println("read: file does not exist")
			return ""
		}

		//read chunk (rpc to chunkserver)
		connection := connect()
		lock(connection, "/"+Chunkid)
		fmt.Println("read lock")
		Data, _ = rpcToNodeReadChunk(location, Chunkid, chunk_offset, newsize)
		unlock(connection, "/"+Chunkid)
		fmt.Println("read unlock")
		close(connection)
	}

	return Data
}

// delete a file
// 在master对chunks加锁
func deletee(Filename string) {
dialLeader:
	fmt.Println("delete")
	connection := connect()
	port, portErr := get(connection, "/master")
	for portErr == false {
		port, portErr = get(connection, "/master") // 获得当前master的port
	}
	close(connection)

	conn, err := rpc.DialHTTP("tcp", masterLocation+port)
	if err != nil {
		// log.Fatalln("dailing error: ", err)
		fmt.Println("dailing error: ", err)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer conn.Close()

	req := ChunkMasterRequest{Filename, 0, false, false}
	var res ChunkMasterResponse

	err = conn.Call("ChunkMasterHandle.DeleteFileAndChunks", req, &res) // 删除文件
	if err != nil {
		// log.Fatalln("ChunkMasterHandle error: ", err)
		fmt.Println("ChunkMasterHandle error: ", err)
	}
	if res.Errornum == -4 {
		fmt.Println("conntact not leader")
		return
	}
}

// 根据filename和chunkindex找到该file的第chunkindex个chunk在哪个node
func rpcToMasterGetChunkLocation(filename string, chunkindex uint64, write bool, read bool) (string, string, int, []string) {
dialLeader:
	connection := connect()
	port, portErr := get(connection, "/master")
	for portErr == false {
		port, portErr = get(connection, "/master") // 获得当前master的port
	}
	close(connection)

	connmaster, errmaster := rpc.DialHTTP("tcp", masterLocation+port)
	if errmaster != nil {
		// log.Fatalln("dailingmaster error: ", errmaster)
		fmt.Println("dailingmaster error: ", errmaster)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer connmaster.Close()

	reqmaster := ChunkMasterRequest{filename, chunkindex, write, read}
	var resmaster ChunkMasterResponse

	errmaster = connmaster.Call("ChunkMasterHandle.GetChunkLocation", reqmaster, &resmaster) // 得到块位置
	if errmaster != nil {
		// log.Fatalln("ChunkMasterHandle error: ", errmaster)
		fmt.Println("ChunkMasterHandle error: ", errmaster)
	}
	if resmaster.Errornum != 0 {
		fmt.Println("getChunkLocation errornum")
		var tempStrings []string
		return "", "", resmaster.Errornum, tempStrings
	}
	if resmaster.Errornum == -4 {
		fmt.Println("conntact not leader")
		var tempStrings []string
		return "", "", resmaster.Errornum, tempStrings
	}
	return resmaster.Chunklocation, resmaster.Chunkid, resmaster.Errornum, resmaster.AllServerLocation
}

func rpcToMasterAddChunk(filename string, chunkindex uint64) (string, string, int) {
dialLeader:
	connection := connect()
	port, portErr := get(connection, "/master")
	for portErr == false {
		port, portErr = get(connection, "/master") // 获得当前master的port
	}
	close(connection)

	connmaster, errmaster := rpc.DialHTTP("tcp", masterLocation+port)
	if errmaster != nil {
		// log.Fatalln("dailingmaster error: ", errmaster)
		fmt.Println("dailingmaster error: ", errmaster)
		// 这时候是leader死掉了，正在等待新的master，要从头开始
		goto dialLeader
	}
	defer connmaster.Close()

	reqmaster := ChunkMasterRequest{filename, chunkindex, true, false}
	var resmaster ChunkMasterResponse

	errmaster = connmaster.Call("ChunkMasterHandle.AddChunk", reqmaster, &resmaster) // 增加块儿
	if errmaster != nil {
		// log.Fatalln("ChunkMasterHandle error: ", errmaster)
		fmt.Println("ChunkMasterHandle error: ", errmaster)
	}
	if resmaster.Errornum != 0 {
		fmt.Println("addchunk errornum")
		return "", "", resmaster.Errornum
	}
	if resmaster.Errornum == -4 {
		fmt.Println("conntact not leader")
		return "", "", resmaster.Errornum
	}
	return resmaster.Chunklocation, resmaster.Chunkid, resmaster.Errornum
}

// 直接给chunk发rpc
func rpcToNodeWriteChunk(location string, chunkid string, chunkoffset uint64, data string, allServerLocation []string) int {
	connslave, errslave := rpc.DialHTTP("tcp", location)
	if errslave != nil {
		// log.Fatalln("dailing error: ", errslave)
		fmt.Println("dailing error: ", errslave)
	}
	defer connslave.Close()

	reqslave := ChunkRequest{chunkid, "", chunkoffset, chunkoffset + uint64(len(data)) - 1, data, allServerLocation}
	var resslave ChunkResponse

	errslave = connslave.Call("ChunkHandle.Write", reqslave, &resslave) // 写
	if errslave != nil {
		// log.Fatalln("ChunkHandle error: ", errslave)
		fmt.Println("ChunkHandle error: ", errslave)
	}
	return resslave.Errornum
}

func rpcToNodeReadChunk(location string, chunkid string, chunkoffset uint64, size uint64) (string, int) {
	connslave, errslave := rpc.DialHTTP("tcp", location)
	if errslave != nil {
		// log.Fatalln("dailing error: ", errslave)
		fmt.Println("dailing error: ", errslave)
	}
	defer connslave.Close()

	var tempStrings []string
	reqslave := ChunkRequest{chunkid, "", chunkoffset, chunkoffset + size, "", tempStrings}
	var resslave ChunkResponse

	errslave = connslave.Call("ChunkHandle.Read", reqslave, &resslave) // 读
	if errslave != nil {
		// log.Fatalln("ChunkHandle error: ", errslave)
		fmt.Println("ChunkHandle error: ", errslave)
	}
	return resslave.Data, resslave.Errornum
}
