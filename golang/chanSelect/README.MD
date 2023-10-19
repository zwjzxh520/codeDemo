![图片](https://github.com/zwjzxh520/codeDemo/blob/main/resource/mp.png?raw=true)

![图片](https://github.com/zwjzxh520/codeDemo/blob/main/golang/chanSelect/resource/640.png?raw=true)

图片来源：Midjourney

先看结论：

1. chan 用完后，最好记得关闭，防止内存泄漏

2. chan 关闭时，接收方会收到一个带零值的数据

3. chan 的发送次数与读取次数要一致，否则会内存泄漏

4. for...select 有多个 chan 时，每次随机读取一个 chan



问题代码：

```go
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
```

输出结果：
```text
Code Demo 大全:  0
Code Demo 大全:  1
Code Demo 大全:  2
Code Demo 大全:  3
Code Demo 大全:  4
Code Demo 大全:  5
Code Demo 大全:  6
Code Demo 大全:  7
Code Demo 大全:  8
error:  <nil>
```

发现了没？明明是有10个数字的，只读取到了9个。

当然，有可能你看到的结果是对的，这时只需要多运行几次即可，总会出现少于10个的情况

修复代码：

以 「////」 的注释为重要改动

```go
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
```

输出结果：

```text
Code Demo 大全:  0
Code Demo 大全:  1
Code Demo 大全:  2
Code Demo 大全:  3
Code Demo 大全:  4
Code Demo 大全:  5
Code Demo 大全:  6
Code Demo 大全:  7
Code Demo 大全:  8
error:  nil
error:  <nil>
Code Demo 大全:  9
Code Demo 大全: 已读完
```
这段代码在某些情况下会出现多次读取到已 close 的 chan 里的数据，这是什么原因呢？请留言告诉我

参考：

https://go.dev/tour/concurrency/5

https://juejin.cn/post/7033711399041761311

https://www.jianshu.com/p/d24dfbb33781

