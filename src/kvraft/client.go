package kvraft

import (
	"6.5840/labrpc"
	"sync"
	"time"
)
import "crypto/rand"
import "math/big"

type Clerk struct {
	servers   []*labrpc.ClientEnd
	RequestID int64
	ClientID  int64
	LeaderId  int
	mu        sync.Mutex
	// You will have to modify this struct.
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.RequestID = 0
	ck.ClientID = nrand()
	ck.LeaderId = 0
	// You'll have to add code here.
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	args := GetArgs{Key: key, RequestID: ck.nextRequestId(), ClientID: ck.ClientID}
	for {
		reply := GetReply{}
		ok := ck.servers[ck.LeaderId].Call("KVServer.Get", &args, &reply)
		if ok {
			if reply.Err == ErrWrongLeader {
				ck.nextLeaderId()
			} else if reply.Err == ErrTimeout {
			} else if reply.Err == ErrNoKey {
				return reply.Value
			} else {
				return reply.Value
			}
		} else {
			ck.nextLeaderId()
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := PutAppendArgs{Key: key, Value: value, RequestID: ck.nextRequestId(), ClientID: ck.ClientID}
	for {
		reply := PutAppendReply{}
		ok := ck.servers[ck.LeaderId].Call("KVServer."+op, &args, &reply)
		if ok {
			if reply.Err == ErrWrongLeader {
				ck.nextLeaderId()
			} else if reply.Err == ErrTimeout {
			} else {
				return
			}
		} else {
			//fuck bug, it rpc has no response, just change leaderId!
			ck.nextLeaderId()
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}

func (ck *Clerk) nextRequestId() int64 {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	ck.RequestID++
	return ck.RequestID
}

func (ck *Clerk) nextLeaderId() int {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	ck.LeaderId++
	ck.LeaderId = ck.LeaderId % len(ck.servers)
	return ck.LeaderId
}
