// +build ignore

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Create test begin!")
	create("test-02")
	write("test-02", 13, "motherfuckerImthebest")
	result := read("test-02", 16, 8)
	fmt.Println(result)
	//deletee("test-02")
	fmt.Println("Create test end!")
}
