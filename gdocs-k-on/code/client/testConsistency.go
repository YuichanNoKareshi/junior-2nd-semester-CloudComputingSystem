// +build ignore

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Create test begin!")
	create("consistency-test-01")
	write("consistency-test-01", 0, "bon bu jour le")
	result1 := read("consistency-test-01", 0, 8)
	fmt.Println(result1)

	create("consistency-test-02")
	write("consistency-test-02", 0, "dahuo kankan chichu bu chichu")
	result2 := read("consistency-test-02", 0, 20)
	fmt.Println(result2)

	create("consistency-test-03")
	write("consistency-test-03", 0, "abcdefghijklmn")
	result3 := read("consistency-test-03", 0, 10)
	fmt.Println(result3)

	write("consistency-test-01", 7, "yume wo o i tu du ke ta")
	result4 := read("consistency-test-01", 0, 20)
	fmt.Println(result4)

	//deletee("test-02")
	fmt.Println("Create test end!")
}
