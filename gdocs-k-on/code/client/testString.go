// +build ignore

package main

import (
	"fmt"
	"strconv"
)

func main() {
	s := "hello-19"
	c := []byte(s)
	var index int
	for i := 0; i < len(c); i++ {
		if c[i] == '-' {
			index = i
		}
	}
	realint, _ := strconv.Atoi(s[(index + 1):])
	realfilename := s[:index]
	fmt.Println(realint)
	fmt.Println(realfilename)
	var array = [3]int{1, 2, 3}
	fmt.Println(array)
}
