package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// from main.mrsequential.go
// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func CallForTask() *Reply {
	request := Request{RequestType: AskForTask}
	reply := Reply{}
	ok := call("Coordinator.AssignTask", &request, &reply)
	if ok {
		return &reply
	} else {
		return nil
	}
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		reply := CallForTask()
		switch reply.ReplyType {
		case MapTask:
			err := handleMapTask(reply, mapf)
			if err == nil {
				CallForReport(MapSuccess, reply.TaskID)
			} else {
				CallForReport(MapFailed, reply.TaskID)
			}
		case ReduceTask:
			err := handleReduceTask(reply, reducef)
			if err == nil {
				CallForReport(ReduceSuccess, reply.TaskID)
			} else {
				CallForReport(ReduceFailed, reply.TaskID)
			}
		case Wait:
			time.Sleep(time.Second * 10)
		case Shutdown:
			os.Exit(0)
		}
		time.Sleep(time.Second)
	}

}

func CallForReport(req RequestType, id int) {
	request := Request{RequestType: req, TaskID: id}
	call("Coordinator.Report", &request, nil)
}

func handleReduceTask(reply *Reply, reducef func(string, []string) string) error {
	intermediate := []KeyValue{}
	Nmap := reply.IntArray[0]
	for i := 0; i < Nmap; i++ {
		iname := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(reply.TaskID)
		f, err := os.Open(iname)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(f)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		f.Close()
	}

	sort.Sort(ByKey(intermediate))

	// call Reduce on each distinct key in intermediate[],
	// and print the result to mr-out-.
	oname := "mr-out-" + strconv.Itoa(reply.TaskID)
	ofile, _ := os.CreateTemp("", oname+"*")

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}
	os.Rename(ofile.Name(), oname)
	ofile.Close()

	//after reducef, remove the intermediate files
	for i := 0; i < Nmap; i++ {
		iname := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(reply.TaskID)
		err := os.Remove(iname)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleMapTask(reply *Reply, mapf func(string, string) []KeyValue) error {
	intermediate := []KeyValue{}
	filename := reply.Content
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	kva := mapf(filename, string(content))
	intermediate = append(intermediate, kva...)

	sort.Sort(ByKey(intermediate))
	X := reply.TaskID
	Nreduce := reply.IntArray[1]
	oname_prefix := "mr-" + strconv.Itoa(X) + "-"

	buckets := make([][]KeyValue, Nreduce)
	for i := range buckets {
		buckets[i] = []KeyValue{}
	}

	for _, kv := range kva {
		key := kv.Key
		idx := ihash(key) % Nreduce
		buckets[idx] = append(buckets[idx], kv)
	}

	for i := range buckets {
		oname := oname_prefix + strconv.Itoa(i)
		f, err := os.CreateTemp("", oname+"*")
		if err != nil {
			return err
		}

		enc := json.NewEncoder(f)
		for _, kv := range buckets[i] {
			err := enc.Encode(&kv)
			if err != nil {
				return err
			}
		}
		os.Rename(f.Name(), oname)
		f.Close()
	}
	return nil
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
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
