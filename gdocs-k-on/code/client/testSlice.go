// +build ignore

package main

import (
	"fmt"
)

func main() {
	var slice []string
	slice = append(slice, "test")
	fmt.Println(slice)
	slice = append(slice, "test")
	fmt.Println(slice)
}
