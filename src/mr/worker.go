// worker.go
package mr

import "fmt"
import "log"
import "net/rpc"
import (
	"hash/fnv"
	"os"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"sort"
	"errors"
)

//
// MapPhase functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by MapPhase.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// Worker 主线程,循环请求任务以及报告任务
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	for {
		// 请求任务
		reply := ReqTaskReply{}
		reply = reqTask()
		if reply.TaskDone {
			break
		}
		// 执行任务
		err := doTask(mapf, reducef, reply.Task)
		if err != nil {
			reportTask(reply.Task.TaskIndex, false)
		}
		// 报告任务结果
		reportTask(reply.Task.TaskIndex, true)
	}
	return
}

// 请求任务
func reqTask() ReqTaskReply {
	// 声明参数并赋值
	args := ReqTaskArgs{}
	args.WorkerStatus = true

	reply := ReqTaskReply{}

	// RPC调用
	if ok := call("Master.HandleTaskReq", &args, &reply); !ok {
		log.Fatal("请求任务失败...")
	}

	return reply
}

// 报告任务结果
func reportTask(taskIndex int, isDone bool) ReportTaskReply {
	// 声明参数并赋值
	args := ReportTaskArgs{}
	args.IsDone = isDone
	args.TaskIndex = taskIndex
	args.WorkerStatus = true

	reply := ReportTaskReply{}

	// RPC调用
	if ok := call("Master.HandleTaskReport", &args, &reply); !ok {
		log.Fatal("报告任务失败...")
	}
	return reply

}

// 执行任务
func doTask(mapf func(string, string) []KeyValue, reducef func(string, []string) string, task Task) error {
	if task.TaskPhase == MapPhase {
		err := DoMapTask(mapf, task.FileName, task.TaskIndex, task.ReduceNum)
		return err
	} else if task.TaskPhase == ReducePhase {
		err := DoReduceTask(reducef, task.MapNum, task.TaskIndex)
		return err
	} else {
		log.Fatal("请求任务的任务阶段返回值异常...")
		return errors.New("请求任务的任务阶段返回值异常")
	}
	return nil
}

// 执行map任务
func DoMapTask(mapf func(string, string) []KeyValue, fileName string, mapTaskIndex int, reduceNum int) error {

	fmt.Println("开始处理Map任务...")
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("cannot open %v", fileName)
		return err
	}
	// 读取文件内容
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", fileName)
		return err
	}
	file.Close()
	// 输入map程序
	kva := mapf(fileName, string(content))
	for i := 0; i < reduceNum; i++ {
		// 中间输出文件名mr-X-Y
		intermediateFileName := intermediateName(mapTaskIndex, i)
		fmt.Printf("doMap文件名%s创建\n", intermediateFileName)
		// 创建中间输出文件,并存储为JSON格式
		file, _ := os.Create(intermediateFileName)
		enc := json.NewEncoder(file)
		for _, kv := range kva {
			if ihash(kv.Key)%reduceNum == i {
				enc.Encode(&kv)
			}
		}
		file.Close()
	}
	return nil
}

// 执行reduce任务
func DoReduceTask(reducef func(string, []string) string, mapNum int, reduceTaskIndex int) error {
	fmt.Println("开始处理Reduce任务...")
	// map:string->[]string
	res := make(map[string][]string)
	for i := 0; i < mapNum; i++ {
		// 打开中间文件
		intermediateFileName := intermediateName(i, reduceTaskIndex)
		file, err := os.Open(intermediateFileName)
		if err != nil {
			log.Fatalf("cannot open %v", intermediateFileName)
			return err
		}
		// 反序列化JSON格式文件
		dec := json.NewDecoder(file)
		// 读取文件内容
		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil {
				break
			}
			_, ok := res[kv.Key]
			if !ok {
				res[kv.Key] = make([]string, 0)
			}
			res[kv.Key] = append(res[kv.Key], kv.Value)
		}
		file.Close()
	}
	// 提取key值,用于排序
	var keys []string
	for k := range res {
		keys = append(keys, k)
	}
	// key值排序
	sort.Strings(keys)
	outputFileName := outputName(reduceTaskIndex)
	fmt.Printf("doReduce输出%s文件名\n", outputFileName)
	outputFile, _ := os.Create(outputFileName)
	for _, k := range keys {
		output := reducef(k, res[k])
		// 输出reduce的结果到mr-out-X文件中
		fmt.Fprintf(outputFile, "%v %v\n", k, output)
	}
	outputFile.Close()

	return nil
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	//c, err := rpc.DialHTTP("unix", "mr-socket")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

// 中间文件名
func intermediateName(mapTaskIndex int, reduceTaskIndex int) string {

	var fileName string
	fileName = "mr" + "-" + strconv.Itoa(mapTaskIndex) + "-" + strconv.Itoa(reduceTaskIndex)
	return fileName
}

// 输出文件名
func outputName(reduceTaskIndex int) string {
	var fileName string
	fileName = "mr-out-" + strconv.Itoa(reduceTaskIndex)
	return fileName
}
