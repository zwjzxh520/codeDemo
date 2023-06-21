package main

import (
	"fmt"
	"strconv"
	"time"
)

func main() {
	dataChan, errChan := readData()
	receiveData(dataChan, errChan)
}

func readData() (chan string, chan error) {
	dataChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		for i := 0; i < 10; i++ {
			dataChan <- strconv.Itoa(i)
		}
		errChan <- nil
	}()

	return dataChan, errChan
}

func receiveData(dataChan chan string, errChan chan error) {
	stop := false
	for !stop {
		select {
		case d := <-dataChan:
			// 使用 sleep 能使问题更易出现
			time.Sleep(100 * time.Millisecond)
			fmt.Println("Code Demo 大全: ", d)

		case e := <-errChan:
			stop = true
			fmt.Println("error: ", e)
		}
	}
}
