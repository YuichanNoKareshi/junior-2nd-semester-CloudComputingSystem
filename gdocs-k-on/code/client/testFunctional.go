// +build ignore

package main

import (
	"fmt"
)

func main() {
	fmt.Println("开始功能性测试！")
	create("functional-test-01")
	write("functional-test-01", 0, "bon bu jour le")
	result1 := read("functional-test-01", 0, 8)
	fmt.Println(result1)

	create("functional-test-02")
	write("functional-test-02", 0, "dahuo kankan chichu bu chichu")
	result2 := read("functional-test-02", 0, 20)
	fmt.Println(result2)

	create("functional-test-03")
	write("functional-test-03", 0, "abcdefghijklmn")
	result3 := read("functional-test-03", 0, 10)
	fmt.Println(result3)

	write("functional-test-01", 7, "yume wo oi tuduketa")
	result4 := read("functional-test-01", 0, 20)
	fmt.Println(result4)

	create("functional-test-04")
	write("functional-test-04", 0, "soshite koko made kita")
	result5 := read("functional-test-04", 0, 10)
	fmt.Println(result5)

	create("functional-test-05")
	write("functional-test-05", 0, "demo doushite kana")
	result6 := read("functional-test-05", 0, 10)
	fmt.Println(result6)

	write("functional-test-02", 7, "atui namida ga tomaranai")
	result7 := read("functional-test-02", 0, 20)
	fmt.Println(result7)

	deletee("functional-test-03")

	create("functional-test-03")
	write("functional-test-03", 0, "utumuki kaketa toki")
	result8 := read("functional-test-03", 0, 10)
	fmt.Println(result8)

	create("functional-test-06")
	write("functional-test-06", 0, "kimi no kao ga mieta")
	result9 := read("functional-test-06", 0, 10)
	fmt.Println(result9)

	deletee("functional-test-02")

	//deletee("test-02")
	fmt.Println("功能性测试结束！")
}
