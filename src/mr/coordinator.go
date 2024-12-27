package mr

import (
	"errors"
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

type TaskStatus int

const (
	idle TaskStatus = iota
	running
	finished
	failed
)

type MapTaskInfo struct {
	taskID     int
	TaskStatus TaskStatus
	timestamp  int64
}

type ReduceTaskInfo struct {
	TaskStatus TaskStatus
	timestamp  int64
}

type Coordinator struct {
	// Your definitions here.
	MapTasks    map[string]*MapTaskInfo
	ReduceTasks []*ReduceTaskInfo
	NReduce     int
	NMap        int
	mu          sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) Report(request *Request, reply *Reply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch request.RequestType {
	case MapSuccess:
		for _, v := range c.MapTasks {
			if request.TaskID == v.taskID {
				v.TaskStatus = finished
				break
			}
		}
	case MapFailed:
		for _, v := range c.MapTasks {
			if request.TaskID == v.taskID {
				v.TaskStatus = failed
				break
			}
		}
	case ReduceSuccess:
		v := c.ReduceTasks[request.TaskID]
		v.TaskStatus = finished
	case ReduceFailed:
		v := c.ReduceTasks[request.TaskID]
		v.TaskStatus = failed
	}
	return nil
}

func (c *Coordinator) AssignTask(request *Request, reply *Reply) error {
	if request.RequestType != AskForTask {
		return errors.New("not ask for task!")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	map_cnt := 0
	for filename, v := range c.MapTasks {
		if v.TaskStatus == failed || v.TaskStatus == idle || (v.TaskStatus == running && (time.Now().Unix()-v.timestamp > 10)) {
			v.TaskStatus = running
			v.timestamp = time.Now().Unix()

			reply.ReplyType = MapTask
			reply.IntArray = make([]int, 2)
			reply.IntArray[0] = c.NMap
			reply.IntArray[1] = c.NReduce
			reply.TaskID = v.taskID
			reply.Content = filename
			return nil
		} else if v.TaskStatus == finished {
			map_cnt++
		}
	}
	if map_cnt < len(c.MapTasks) {
		reply.ReplyType = Wait
		return nil
	}
	reduce_cnt := 0
	for i, v := range c.ReduceTasks {
		if v.TaskStatus == failed || v.TaskStatus == idle || (v.TaskStatus == running && (time.Now().Unix()-v.timestamp > 10)) {
			v.TaskStatus = running
			v.timestamp = time.Now().Unix()

			reply.ReplyType = ReduceTask
			reply.IntArray = make([]int, 2)
			reply.IntArray[0] = c.NMap
			reply.IntArray[1] = c.NReduce
			reply.TaskID = i
			return nil
		} else if v.TaskStatus == finished {
			reduce_cnt++
		}
	}
	if reduce_cnt < len(c.ReduceTasks) {
		reply.ReplyType = Wait
		return nil
	}
	reply.ReplyType = Shutdown
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	for _, task := range c.MapTasks {
		if task.TaskStatus != finished {
			return false
		}
	}
	for _, task := range c.ReduceTasks {
		if task.TaskStatus != finished {
			return false
		}
	}

	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		MapTasks:    make(map[string]*MapTaskInfo),
		ReduceTasks: make([]*ReduceTaskInfo, nReduce),
		NReduce:     nReduce,
		NMap:        len(files),
	}
	for i, file := range files {
		c.MapTasks[file] = &MapTaskInfo{
			taskID:     i,
			TaskStatus: idle,
		}
	}
	for i := 0; i < nReduce; i++ {
		c.ReduceTasks[i] = &ReduceTaskInfo{
			TaskStatus: idle,
		}
	}
	c.server()
	return &c
}
