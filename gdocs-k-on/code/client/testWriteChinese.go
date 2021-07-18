// +build ignore

package main

import (
	"fmt"
)

func test1() {
	create("test-01")
	write("test-01", 0, "bon不jour了大伙")
	result1 := read("test-01", 0, 10)
	fmt.Println(result1)

	//deletee("test-02")
	fmt.Println("Create test end!")
}
func test2() {
	create("test-01")
	write("test-01",3,"测测中文到底占几个字符! 你妈的")
	//write("test-01",1024*1024-10,"abcdefghijklmnopqrstuvwxyz")
	result2 := read("test-01",3,50)
	fmt.Println(result2)
}
func test3() {
	s := "abcdefghijklmnopqrstuvwxyz"
	tempdata := s[0 : 0+10] // 会取从lastoffset到lastoffset+newsize共newsize个字符
	leftdata := s[10 : len(s)]
	fmt.Println(tempdata)
	fmt.Println(leftdata)
}

func main() {
	fmt.Println("Chinese written test begin!")
	test2()
	
}
