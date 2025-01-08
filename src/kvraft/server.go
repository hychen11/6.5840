package kvraft

import (
	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const HandleOpTimeOut = 500 * time.Millisecond
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type OpType int

const (
	OpGet OpType = iota
	OpPut
	OpAppend
)

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	OpType    OpType
	Key       string
	Value     string
	RequestID int64
	ClientID  int64
}

type ReplyState struct {
	LastRequestId int64
	LastReply     *OpReply
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	client     map[int64]ReplyState //key clientId, value requestId
	db         map[string]string
	commitChan map[int]chan *OpReply //key startIndex, value result
}

type OpReply struct {
	RequestId int64
	Err       Err
	Value     string
}

func (kv *KVServer) HandleOp(args *Op, reply *OpReply) {
	//check if duplicated
	kv.mu.Lock()
	state, exist := kv.client[args.ClientID]
	if args.OpType != OpGet && exist && args.RequestID <= state.LastRequestId {
		DPrintf("HandleOp: server %v have duplicate operation on Put & Append: args.RequestID <= state.LastRequestId", kv.me)
		reply = state.LastReply
		kv.mu.Unlock()
		return
	}
	kv.mu.Unlock()

	index, _, isLeader := kv.rf.Start(*args)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.mu.Lock()
	if _, ok := kv.commitChan[index]; !ok {
		DPrintf("HandleOp: commitChann %v not exists", index)
		kv.commitChan[index] = make(chan *OpReply)
	}
	ch := kv.commitChan[index]
	kv.mu.Unlock()

	defer func() {
		kv.mu.Lock()
		close(ch)
		delete(kv.commitChan, index)
		kv.mu.Unlock()
	}()

	select {
	case result := <-ch:
		reply.Value = result.Value
		reply.Err = result.Err
		return
	case <-time.After(HandleOpTimeOut):
		reply.Err = ErrTimeout
		return
	}

}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := Op{OpType: OpGet, RequestID: args.RequestID, ClientID: args.ClientID, Key: args.Key}
	res := OpReply{}
	kv.HandleOp(&opArgs, &res)
	reply.Err = res.Err
	reply.Value = res.Value
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := Op{OpType: OpPut, RequestID: args.RequestID, ClientID: args.ClientID, Key: args.Key, Value: args.Value}
	res := OpReply{}
	kv.HandleOp(&opArgs, &res)
	reply.Err = res.Err
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := Op{OpType: OpAppend, RequestID: args.RequestID, ClientID: args.ClientID, Key: args.Key, Value: args.Value}
	res := OpReply{}
	kv.HandleOp(&opArgs, &res)
	reply.Err = res.Err
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// to apply the committed entries to state machine!
func (kv *KVServer) applier() {
	for !kv.killed() {
		msg := <-kv.applyCh
		DPrintf("applier receive apply messahe %v", msg)
		if msg.CommandValid {
			kv.mu.Lock()

			//first check the msg has latest snapshot (4B)

			//second turn msg's command into struct Op
			op := msg.Command.(Op)
			requestId := op.RequestID
			clientId := op.ClientID
			//check if it is old requset
			var res *OpReply
			state, exist := kv.client[clientId]
			if op.OpType != OpGet && exist && requestId <= state.LastRequestId {
				//old just return old result directly
				res = state.LastReply
			} else {
				// if new just apply to statemachine!
				res = kv.DBExecute(&op)
				if op.OpType != OpGet {
					//record reply
					kv.client[clientId] = ReplyState{LastRequestId: requestId, LastReply: res}
				}
			}
			_, isLeader := kv.rf.GetState()
			ch, exist := kv.commitChan[msg.CommandIndex]
			kv.mu.Unlock()

			if isLeader && exist {
				ch <- res
			} else if !isLeader {
				DPrintf("Not leader, no need to send commitChan")
			} else {
				DPrintf("leader %v  find commitChan not exist, may timeout closed", kv.me)
			}
		}
	}
}

func (kv *KVServer) DBExecute(op *Op) *OpReply {
	reply := &OpReply{}
	reply.RequestId = op.RequestID
	switch op.OpType {
	case OpGet:
		if value, exist := kv.db[op.Key]; exist {
			reply.Value = value
			reply.Err = OK
		} else {
			reply.Value = ""
			reply.Err = ErrNoKey
		}
	case OpPut:
		kv.db[op.Key] = op.Value
		reply.Err = OK
	case OpAppend:
		val, exist := kv.db[op.Key]
		if !exist {
			kv.db[op.Key] = op.Value
		} else {
			kv.db[op.Key] = val + op.Value
		}
		reply.Err = OK
	}
	return reply
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	kv.client = make(map[int64]ReplyState)
	kv.db = make(map[string]string)
	kv.commitChan = make(map[int]chan *OpReply)

	go kv.applier()
	return kv
}
