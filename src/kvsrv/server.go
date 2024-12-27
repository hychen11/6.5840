package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

type ReplyState struct {
	LastRequestId int64
	LastReply     string
}

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type KVServer struct {
	mu     sync.Mutex
	kvmap  map[string]string
	client map[int64]ReplyState
	// Your definitions here.
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	value, exist := kv.kvmap[args.Key]
	if exist {
		reply.Value = value
	} else {
		reply.Value = ""
	}
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.kvmap[args.Key] = args.Value
	return
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	requestId := args.RequestID
	clientId := args.ClientID

	state, exist := kv.client[clientId]
	if exist && requestId <= state.LastRequestId {
		reply.Value = state.LastReply
		return
	} else {
		value, exist := kv.kvmap[args.Key]
		if exist {
			reply.Value = value
			kv.kvmap[args.Key] = value + args.Value
		} else {
			reply.Value = ""
			kv.kvmap[args.Key] = args.Value
		}
	}
	kv.client[clientId] = ReplyState{LastRequestId: requestId, LastReply: reply.Value}
	return
}

func StartKVServer() *KVServer {
	kv := KVServer{
		kvmap:  make(map[string]string),
		client: make(map[int64]ReplyState),
	}

	// You may need initialization code here.

	return &kv
}
