package main

import (
	"fmt"
	"strconv"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(10)
	fmt.Println("multi read test begin")
	prefix := "test-"
	for i := 0; i < 10; i++ {
		filename := prefix + strconv.Itoa(i)
		go func() {
			read(filename, 0, 200)
			// fmt.Println("From ", filename, " we read ", result)
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("multi read test end")
}
