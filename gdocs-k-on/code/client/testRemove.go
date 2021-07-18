// +build ignore

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Remove test begin!")
	delete("test-01")
	delete("test-02")
}
