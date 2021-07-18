// +build ignore

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Create test begin!")
	create("consistency2-test-01")
	write("consistency2-test-01", 0, "xiangwanxiangwanxiangwan")
	result1 := read("consistency2-test-01", 0, 8)
	fmt.Println(result1)

	create("consistency2-test-02")
	write("consistency2-test-02", 0, "bzhanguanzhujiaranjintainchishenme")
	result2 := read("consistency2-test-02", 0, 20)
	fmt.Println(result2)

	create("consistency2-test-03")
	write("consistency2-test-03", 0, "leidayaoquankai")
	result3 := read("consistency2-test-03", 0, 10)
	fmt.Println(result3)

	write("consistency2-test-01", 7, "wojiehsounidetiaozhan")
	result4 := read("consistency2-test-01", 0, 20)
	fmt.Println(result4)

	//deletee("test-02")
	fmt.Println("Create test end!")
}
