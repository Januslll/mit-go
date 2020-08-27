// master.go
package mr

import "log"
import "net"
import "net/rpc"
import (
	"net/http"
	"sync"
	"time"
	"fmt"
	"os"
	"errors"
)

// 任务状态定义
type TaskState struct {
	// 状态
	Status TaskStatus
	// 开始执行时间
	StartTime time.Time
}

// Master结构定义
type Master struct {
	// 任务队列
	TaskChan chan Task
	// 输入文件
	Files []string
	// map数目
	MapNum int
	// reduce数目
	ReduceNum int
	// 任务阶段
	TaskPhase TaskPhase
	// 任务状态
	TaskState []TaskState
	// 互斥锁
	Mutex sync.Mutex
	// 是否完成
	IsDone bool
}

// 启动Master
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// 初始化Master
	m.IsDone = false
	m.Files = files
	m.MapNum = len(files)
	m.ReduceNum = nReduce
	m.TaskPhase = MapPhase
	m.TaskState = make([]TaskState, m.MapNum)
	m.TaskChan = make(chan Task, 10)
	for k := range m.TaskState {
		m.TaskState[k].Status = TaskStatusReady
	}

	// 开启线程监听
	m.server()

	return &m
}

// 启动一个线程监听worker.go的RPC请求
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", "127.0.0.1:1234")
	os.Remove("mr-socket")
	l, e := net.Listen("unix", "mr-socket")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// 处理任务请求
func (m *Master) HandleTaskReq(args *ReqTaskArgs, reply *ReqTaskReply) error {
	fmt.Println("开始处理任务请求...")
	if !args.WorkerStatus {
		return errors.New("当前worker已下线")
	}
	// 任务出队列
	task, ok := <-m.TaskChan
	if ok == true {
		reply.Task = task
		// 任务状态置为执行中
		m.TaskState[task.TaskIndex].Status = TaskStatusRunning
		// 记录任务开始执行时间
		m.TaskState[task.TaskIndex].StartTime = time.Now()
	} else {
		// 若队列中已经没有任务，则任务全部完成，结束
		reply.TaskDone = true
	}
	return nil
}

// 处理任务报告
func (m *Master) HandleTaskReport(args *ReportTaskArgs, reply *ReportTaskReply) error {
	fmt.Println("开始处理任务报告...")
	if !args.WorkerStatus {
		reply.MasterAck = false
		return errors.New("当前worker已下线")
	}
	if args.IsDone == true {
		// 任务已完成
		m.TaskState[args.TaskIndex].Status = TaskStatusFinish
	} else {
		// 任务执行错误
		m.TaskState[args.TaskIndex].Status = TaskStatusErr
	}
	reply.MasterAck = true
	return nil
}

// 循环调用 Done() 来判定任务是否完成
func (m *Master) Done() bool {
	ret := false

	finished := true
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	for key, ts := range m.TaskState {
		switch ts.Status {
		case TaskStatusReady:
			// 任务就绪
			finished = false
			m.addTask(key)
		case TaskStatusQueue:
			// 任务队列中
			finished = false
		case TaskStatusRunning:
			// 任务执行中
			finished = false
			m.checkTask(key)
		case TaskStatusFinish:
			// 任务已完成
		case TaskStatusErr:
			// 任务错误
			finished = false
			m.addTask(key)
		default:
			panic("任务状态异常...")
		}
	}
	// 任务完成
	if finished {
		// 判断阶段
		// map则初始化reduce阶段
		// reduce则结束
		if m.TaskPhase == MapPhase {
			m.initReduceTask()
		} else {
			m.IsDone = true
			close(m.TaskChan)
		}
	} else {
		m.IsDone = false
	}
	ret = m.IsDone
	return ret
}

// 初始化reduce阶段
func (m *Master) initReduceTask() {
	m.TaskPhase = ReducePhase
	m.IsDone = false
	m.TaskState = make([]TaskState, m.ReduceNum)
	for k := range m.TaskState {
		m.TaskState[k].Status = TaskStatusReady
	}
}

// 将任务放入任务队列中
func (m *Master) addTask(taskIndex int) {
	// 构造任务信息
	m.TaskState[taskIndex].Status = TaskStatusQueue
	task := Task{
		FileName:  "",
		MapNum:    len(m.Files),
		ReduceNum: m.ReduceNum,
		TaskIndex: taskIndex,
		TaskPhase: m.TaskPhase,
		IsDone:    false,
	}
	if m.TaskPhase == MapPhase {
		task.FileName = m.Files[taskIndex]
	}
	// 放入任务队列
	m.TaskChan <- task
}

// 检查任务处理是否超时
func (m *Master) checkTask(taskIndex int) {
	timeDuration := time.Now().Sub(m.TaskState[taskIndex].StartTime)
	if timeDuration > MaxTaskRunTime {
		// 任务超时重新加入队列
		m.addTask(taskIndex)
	}
}
