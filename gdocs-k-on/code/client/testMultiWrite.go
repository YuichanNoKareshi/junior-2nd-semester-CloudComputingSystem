package main

import (
	"fmt"
	"strconv"
	"sync"
)

func main() {
	fmt.Println("multi write test begin")
	wg := sync.WaitGroup{}
	prefix := "test-"
	for i := 0; i < 1; i++ {
		filename := prefix + strconv.Itoa(i)
		wg.Add(10)
		for j := 0; j < 10; j++ {
			temp := j
			go func() {
				content := "nanaminanami"
				write(filename, uint64(temp*15), content)
				wg.Done()
			}()
		}
		wg.Wait()
	}

	fmt.Println("multi write test end")
}
