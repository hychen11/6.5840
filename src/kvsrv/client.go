package kvsrv

import (
	"6.5840/labrpc"
	"sync"
	"time"
)
import "crypto/rand"
import "math/big"

type Clerk struct {
	server    *labrpc.ClientEnd
	RequestID int64
	ClientID  int64
	mu        sync.Mutex
	// You will have to modify this struct.
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := Clerk{
		server:    server,
		RequestID: 0,
		ClientID:  nrand(),
	}
	// You'll have to add code here.
	return &ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	args := GetArgs{Key: key}
	reply := GetReply{}
	for {
		ok := ck.server.Call("KVServer.Get", &args, &reply)
		if ok {
			return reply.Value
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return ""
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// You will have to modify this function.
	if op == "Append" {
		args := PutAppendArgs{Key: key, Value: value, RequestID: ck.nextRequestId(), ClientID: ck.ClientID}
		reply := PutAppendReply{}
		for {
			ok := ck.server.Call("KVServer."+op, &args, &reply)
			if ok {
				return reply.Value
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	} else {
		args := PutAppendArgs{Key: key, Value: value}
		reply := PutAppendReply{}
		for {
			ok := ck.server.Call("KVServer."+op, &args, &reply)
			if ok {
				return ""
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	return ""
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}

func (ck *Clerk) nextRequestId() int64 {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	ck.RequestID++
	return ck.RequestID
}
