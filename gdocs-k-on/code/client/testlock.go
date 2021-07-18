// +build ignore

package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

func test() {
		// fmt.Println("test")
		// connection := connect()
		// fmt.Println(strconv.Itoa(i)+" connect success")
		// lock(connection,"/test-lock")
		// fmt.Println(strconv.Itoa(i)+" lock success")
		// unlock(connection,"/test-lock")
		// fmt.Println(strconv.Itoa(i)+" unlock success")
		// close(connection)
		// wg.Done()

		wg :=sync.WaitGroup{}
		wg.Add(10)
		fmt.Println("test lock")
		for i := 0; i < 10; i++ {
			temp := i
			go func() {
				fmt.Println(strconv.Itoa(temp)+"test")
				connection := connect()
				fmt.Println(strconv.Itoa(temp)+" connect success")
				lock(connection,"/test-lock")
				fmt.Println(strconv.Itoa(temp)+" lock success")
				time.Sleep(time.Second * 3)
				unlock(connection,"/test-lock")
				fmt.Println(strconv.Itoa(temp)+" unlock success")
				close(connection)
				wg.Done()
			}()
		}
		wg.Wait()
}

func main() {
	// wg := sync.WaitGroup{}
	// wg.Add(3)
	// fmt.Println("test lock")
	// go func() {
	// 	i:=1
	// 	fmt.Println(strconv.Itoa(i)+"test")
	// 	connection := connect()
	// 	fmt.Println(strconv.Itoa(i)+" connect success")
	// 	lock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" lock success")
	// 	time.Sleep(time.Second * 3)
	// 	unlock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" unlock success")
	// 	close(connection)
	// 	wg.Done()
	// }()
	// go func() {
	// 	i:=2
	// 	fmt.Println(strconv.Itoa(i)+"test")
	// 	connection := connect()
	// 	fmt.Println(strconv.Itoa(i)+" connect success")
	// 	lock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" lock success")
	// 	time.Sleep(time.Second * 3)
	// 	unlock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" unlock success")
	// 	close(connection)
	// 	wg.Done()
	// }()
	// go func() {
	// 	i:=3
	// 	fmt.Println(strconv.Itoa(i)+"test")
	// 	connection := connect()
	// 	fmt.Println(strconv.Itoa(i)+" connect success")
	// 	lock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" lock success")
	// 	time.Sleep(time.Second * 1)
	// 	unlock(connection,"/test-lock")
	// 	fmt.Println(strconv.Itoa(i)+" unlock success")
	// 	close(connection)
	// 	wg.Done()
	// }()
	// wg.Wait()
	test()
}
