package main

import (
	"fmt"
	"github.com/go-zookeeper/zk"
	"time"
)

//访问权限为All
var acls = zk.WorldACL(zk.PermAll)

//这一部分为工具函数
func get(conn *zk.Conn, path string) (string, bool) {
	data, _, err := conn.Get(path)
	if err != nil {
		fmt.Printf("查询%s失败, err: %v\n", path, err)
		return "", false
	}
	stringData := string(data)
	return stringData, true
}

func exists(conn *zk.Conn, path string) (bool, bool) {
	result, _, err := conn.Exists(path)
	if err != nil {
		fmt.Printf("存在%s失败, err: %v\n", path, err)
		return false, true
	}
	return result, false
}

func existw(conn *zk.Conn, path string) (bool, bool, <-chan zk.Event) {
	result, _, event, err := conn.ExistsW(path)
	if err != nil {
		fmt.Printf("存在%s失败, err: %v\n", path, err)
		return false, true, event
	}
	return result, false, event
}

func remove(conn *zk.Conn, path string) bool {
	err := conn.Delete(path, 0)
	if err != nil {
		fmt.Printf("删除%s失败, err: %v\n", path, err)
		return true
	}
	return false
}

func add(conn *zk.Conn, path string, data string) bool {
	var dataslice = []byte(data)
	// flags有4种取值：
	// 0:永久，除非手动删除
	// zk.FlagEphemeral = 1:短暂，session断开则该节点也被删除
	// zk.FlagSequence  = 2:会自动在节点后面添加序号
	// 3:Ephemeral和Sequence，即，短暂且自动添加序号
	var flags int32 = 1
	_, err := conn.Create(path, dataslice, flags, acls)
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return false
	}
	return true
}

func connect() *zk.Conn {
	hosts := []string{"123.60.25.41:2181"}
	// 连接zk
retry:
	connzk, _, errzk := zk.Connect(hosts, time.Second*60)
	if errzk != nil {
		fmt.Println("error:")
		fmt.Println(errzk)
		goto retry
	}
	return connzk
}

func close(conn *zk.Conn) {
	conn.Close()
}

//在使用前应当open，而且应当记得close
//这一部分为真正的lock函数
func lock(conn *zk.Conn, chunkid string) {
retrylock:
	exist, err := exists(conn, chunkid)
	for err {
		exist, err = exists(conn, chunkid)
	}
	if !exist {
		fmt.Println("nnnnnnnnnnnotexist")
		addornot := add(conn, chunkid, "lock")
		if !addornot {
			goto retrylock
		} // 如果addornot成则返回
	} else {
		fmt.Println("eeeeeeeeeeeexist")
		existW, errW, event := existw(conn, chunkid)
		for errW {
			existW, errW, event = existw(conn, chunkid)
		}
		if !existW {
			fmt.Println("wwwwwwwwnnnnnotexist")
			goto retrylock
		} else {
			fmt.Println("wwwwwwwwexist")
			//existW为true该怎么办
			watchCreataNode(event)
			goto retrylock
		}
	}
}

func lockmaster(conn *zk.Conn, port string, chRetry chan bool, chGetLock chan bool) {
retrylock:
	exist, err := exists(conn, "/master")
	for err {
		exist, err = exists(conn, "/master")
	}
	if !exist {
		fmt.Println("notexist")
		addornot := add(conn, "/master", port)
		if !addornot {
			goto retrylock
		} else {
			chGetLock <- true // 自己拿到锁
			return
		}
	} else {
		existW, errW, event := existw(conn, "/master")
		for errW {
			existW, errW, event = existw(conn, "/master")
		}
		if !existW {
			fmt.Println("wnotexist")
			goto retrylock
		} else {
			//existW为true该怎么办
			chGetLock <- false // 别人拿到锁了

			<-event // leader挂了

			chRetry <- true // 可以retry了
			// goto retrylock
		}
	}
}

func unlock(conn *zk.Conn, chunkid string) {
	exist, err := exists(conn, chunkid)
	for err {
		exist, err = exists(conn, chunkid)
	}
	if exist {
		remove(conn, chunkid)
	}
}

func watchCreataNode(ech <-chan zk.Event) {
	event := <-ech
	fmt.Println("*******************")
	fmt.Println("path:", event.Path)
	fmt.Println("type:", event.Type.String())
	fmt.Println("state:", event.State.String())
	fmt.Println("-------------------")
}

// func watchMaster(ech <-chan zk.Event, port string, chRetry chan bool) {
// 	event := <-ech
// 	fmt.Println("leader is dead")
// 	fmt.Println("path:", event.Path)

// 	//port rpc

// }
