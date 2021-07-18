// +build ignore

package main


import (
    "fmt"
	"github.com/go-zookeeper/zk"
    "time"
)

//访问权限为All
var acls=zk.WorldACL(zk.PermAll)
//一直不断尝试直到连接上为止

func get(conn *zk.Conn,path string) string {
	data, _, err := conn.Get(path)
	if err != nil {
		fmt.Printf("查询%s失败, err: %v\n", path, err)
		return ""
	}
    stringData := string(data)
    return stringData
}

func exist(conn *zk.Conn,path string) (bool,bool) {
	result, _, err := conn.Exists(path)
	if err != nil {
		fmt.Printf("存在%s失败, err: %v\n", path, err)
		return false,true
	}
	return result,false
}

func remove(conn *zk.Conn,path string) {
	err := conn.Delete(path,0)
	if err != nil {
		fmt.Printf("删除%s失败, err: %v\n", path, err)
		return
	}
    return
}

func add(conn *zk.Conn,path string,data string) {
	var dataslice = []byte(data)
	// flags有4种取值：
	// 0:永久，除非手动删除
	// zk.FlagEphemeral = 1:短暂，session断开则该节点也被删除
	// zk.FlagSequence  = 2:会自动在节点后面添加序号
	// 3:Ephemeral和Sequence，即，短暂且自动添加序号
	var flags int32 = 1
	s, err := conn.Create(path, dataslice, flags, acls)
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return
	}
	fmt.Printf("创建: %s 成功", s)
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

// func main() {
// 	// hosts := []string{"123.60.25.41:2181"}
// 	// // 连接zk
// 	// connzk, _, errzk := zk.Connect(hosts, time.Second*60)
// 	// defer connzk.Close()
// 	// if errzk != nil {
// 	// 	fmt.Println("error:")
//     //     fmt.Println(errzk)
// 	// 	return
// 	// }
// 	// fmt.Println(connzk.Server())
// 	// get(connzk,"/test2")
// 	connection := connect()
// 	get(connection,"/test2")
// 	close(connection)
// }