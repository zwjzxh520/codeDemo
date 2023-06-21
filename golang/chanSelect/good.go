package main

import (
	"errors"
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
		//// close chan
		//// 切记，关闭所有不再使用的 chan
		defer func() {
			close(dataChan)
			close(errChan)
		}()

		for i := 0; i < 10; i++ {
			dataChan <- strconv.Itoa(i)
		}
		//// 实际当中如没有错误，可以不用强行写入错误，直接 close 即可。此处是为演示用
		errChan <- errors.New("nil")
	}()

	return dataChan, errChan
}

func receiveData(dataChan chan string, errChan chan error) {
	//// 确保两个 chan 的数据均读完，防止内存泄漏
	dataDone := false
	errDone := false
	for !dataDone || !errDone {
		select {
		case d := <-dataChan:
			//// 判断 chan 是否关闭
			if d == "" {
				fmt.Println("Code Demo 大全: 已读完")
				dataDone = true
				break
			}

			// 使用 sleep 能使问题更易出现
			time.Sleep(100 * time.Millisecond)
			fmt.Println("Code Demo 大全: ", d)

		case e := <-errChan:
			fmt.Println("error: ", e)
			//// 判断 chan 是否关闭
			if e == nil {
				errDone = true
				break
			}
		}
	}
}
