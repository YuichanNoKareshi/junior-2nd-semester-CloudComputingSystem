// +build ignore

package main

import "fmt"


func main() {
	fmt.Println("testzookeepermain")
	connection := connect()
	fmt.Println(get(connection,"/test2"))
	fmt.Println(get(connection,"/test3"))
	fmt.Println(get(connection,"/test2"))
	fmt.Println(exist(connection,"/test2"))
	add(connection,"/test3","test")
	fmt.Println(exist(connection,"/test3"))
	fmt.Println(get(connection,"/test3"))
	remove(connection,"/test3")
	close(connection)
}