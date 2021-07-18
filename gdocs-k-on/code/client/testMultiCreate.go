package main

import (
	"fmt"
	"strconv"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(10)
	fmt.Println("multi create test begin")
	prefix := "test-"
	for i := 0; i < 10; i++ {
		filename := prefix + strconv.Itoa(i)
		go func() {
			create(filename)
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("multi create test end")
}
