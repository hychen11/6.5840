package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

type RequestType int

const (
	AskForTask RequestType = iota
	MapSuccess
	MapFailed
	ReduceSuccess
	ReduceFailed
)

type ReplyType int

const (
	MapTask ReplyType = iota
	ReduceTask
	Wait
	Shutdown
)

type Request struct {
	RequestType RequestType
	TaskID      int
	IntArray    []int
	Content     string
}

type Reply struct {
	ReplyType ReplyType
	TaskID    int
	IntArray  []int
	Content   string
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
