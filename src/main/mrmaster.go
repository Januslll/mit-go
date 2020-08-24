// mrmaster.go
package main

//
// start the master process, which is implemented
// in ../mr/master.go
//
// go run mrmaster.go pg*.txt
//

import "mr"
import "time"
import "os"
import "fmt"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: mrmaster inputfiles...\n")
		os.Exit(1)
	}

	// 调用master.go里实现的MakeMaster方法，生产一个master
	// 入参1：输入文件
	// 入参2：map任务阶段，需要将中间结果集分割成10份作为reduce的入参
	m := mr.MakeMaster(os.Args[1:], 10)

	// 调用master.go的Done方法，如果没有完成则继续等待
	for m.Done() == false {
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)
}
