// rpc.go
package mr

import "time"

//
// RPC 定义
//

type TaskStatus string
type TaskPhase string
type TimeDuration time.Duration

// 任务状态常量
const (
	// 就绪
	TaskStatusReady   TaskStatus = "ready"
	// 队列中
	TaskStatusQueue   TaskStatus = "queue"
	// 执行中
	TaskStatusRunning TaskStatus = "running"
	// 已完成
	TaskStatusFinish  TaskStatus = "finish"
	// 任务错误
	TaskStatusErr     TaskStatus = "error"
)

//任务阶段常量
const (
	MapPhase    TaskPhase = "map"
	ReducePhase TaskPhase = "reduce"
)

// 时间常量
const (
	Nanosecond     time.Duration = 1
	Microsecond                  = 1000 * Nanosecond
	Millisecond                  = 1000 * Microsecond
	Second                       = 1000 * Millisecond
	Minute                       = 60 * Second
	Hour                         = 60 * Minute
	MaxTaskRunTime               = 10 * Second
)

// 任务定义
type Task struct {
	// 操作阶段：map/reduce
	TaskPhase TaskPhase
	// map个数
	MapNum int
	// reduce个数
	ReduceNum int
	// 任务序号
	TaskIndex int
	// 文件名
	FileName string
	// 是否完成
	IsDone bool
}

// 请求任务参数
type ReqTaskArgs struct {
	// 当前worker存活,可以执行任务
	WorkerStatus bool
}

// 请求任务返回值
type ReqTaskReply struct {
	// 返回一个任务
	Task Task
	// 是否完成所有任务
	TaskDone bool
}

// 报告任务参数
type ReportTaskArgs struct {
	// 当前worker存活,可以执行任务
	WorkerStatus bool
	// 任务序号
	TaskIndex int
	// 是否完成
	IsDone bool
}

// 报告任务返回值
type ReportTaskReply struct {
	// master响应是否处理成功
	MasterAck bool
}
