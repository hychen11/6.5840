package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"6.5840/labgob"
	"bytes"
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

type StateType int

const (
	ElectionTimeout  = 150 //random election timeouts 150-300ms, to avoid elect crash!
	HeartBeatTimeout = 101
)

const (
	leader StateType = iota
	follower
	candidate
)

type Entry struct {
	Term int
	Cmd  interface{}
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	//persist state
	currentTerm int
	votedFor    int
	log         []Entry
	//Volatile state
	commitIndex int
	lastApplied int
	//Volatile state on leaders
	//(Reinitialized after election)
	nextIndex  []int
	matchIndex []int
	applyCh    chan ApplyMsg //with no buffer

	timestamp time.Time
	state     StateType
	//vote
	muVote  sync.Mutex
	voteCnt int
	//channel
	cond *sync.Cond
	//snapshot
	lastIncludedIndex int //the index of the last entry in the log that the snapshot replaces
	lastIncludedTerm  int //the term of this entry
	snapshot          []byte
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	heartTimer *time.Timer
}

//Upon election: send initial empty AppendEntries RPCs (heartbeat) to each server; repeat during idle periods to prevent election timeouts (§5.2)
//• If command received from client: append entry to local log, respond after entry applied to state machine (§5.3)
//• If last log index ≥ nextIndex for a follower: send AppendEntries RPC with log entries starting at nextIndex
//• If successful: update nextIndex and matchIndex for follower (§5.3)
//• If AppendEntries fails because of log inconsistency: decrement nextIndex and retry (§5.3)
//• If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm: set commitIndex = N (§5.3, §5.4).

// create a goroutine with a loop that calls time.Sleep()
func (rf *Raft) SendHeartbeat() {
	for !rf.killed() {
		<-rf.heartTimer.C

		rf.mu.Lock()
		if rf.state != leader {
			rf.mu.Unlock()
			return
		}
		////leader nextIndex and matchIndex can be the latest log
		//rf.nextIndex[rf.me] = len(rf.log)
		//rf.matchIndex[rf.me] = rf.nextIndex[rf.me] - 1

		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: rf.nextIndex[i] - 1,
				//PrevLogTerm:  rf.log[rf.nextIndex[i]-1].Term,
				LeaderCommit: rf.commitIndex,
			}
			//dont have log, then leader need to call TnstallSnapshot to follwer to make up the snapshot and log
			if args.PrevLogIndex < rf.lastIncludedIndex {
				go rf.handleInstallSnapshot(i)
			} else {
				if len(rf.log)-1+rf.lastIncludedIndex >= rf.nextIndex[i] {
					//append Entry
					args.Entries = rf.log[rf.nextIndex[i]-rf.lastIncludedIndex:]
				} else {
					//send heartbeat
					args.Entries = make([]Entry, 0)
				}
				args.PrevLogTerm = rf.log[rf.nextIndex[i]-1-rf.lastIncludedIndex].Term
				go rf.HandleAppendEntries(i, &args)
			}
		}
		rf.mu.Unlock()
		//time.Sleep(time.Duration(HeartBeatTimeout) * time.Millisecond)
		rf.heartTimer.Reset(time.Duration(HeartBeatTimeout) * time.Millisecond)

	}
}

func (rf *Raft) HandleAppendEntries(server int, args *AppendEntriesArgs) {
	reply := &AppendEntriesReply{}
	ok := rf.SendAppendEntries(server, args, reply)
	if !ok {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term != rf.currentTerm {
		return
	}
	//still a leader
	//If successful: update nextIndex and matchIndex for follower (§5.3)
	if reply.Success {
		//rf.nextIndex[server] += len(args.Entries)
		//rf.matchIndex[server] = rf.nextIndex[server] - 1
		rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
		rf.nextIndex[server] = rf.matchIndex[server] + 1
		//DPrintf("rf.nextIndex[%v] %v rf.matchIndex[%v] %v", server, rf.nextIndex[server], server, rf.matchIndex[server])
		//If there exists an N such that N > commitIndex, a majority
		//of matchIndex[i] ≥ N, and log[N].term == currentTerm: set commitIndex = N (§5.3, §5.4).
		//bug1: why it cannot count for itself? because leader dont update it's matchIndex and nextIndex
		for N := len(rf.log) - 1 + rf.lastIncludedIndex; N > rf.commitIndex; N-- {
			cnt := 1
			for j := 0; j < len(rf.peers); j++ {
				if j == rf.me {
					continue
				}
				if rf.matchIndex[j] >= N && rf.log[N-rf.lastIncludedIndex].Term == rf.currentTerm {
					cnt++
				}
			}
			if cnt > len(rf.peers)/2 {
				rf.commitIndex = N
				DPrintf("get commitIndex: %v", rf.commitIndex)
				rf.cond.Signal()
				break
			}
		}

		return
	}
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		rf.state = follower
		rf.timestamp = time.Now()
		rf.persist()
		return
	}
	if reply.Term == rf.currentTerm && rf.state == leader {
		//Case 1: leader doesn't have XTerm:
		//nextIndex = XIndex
		//Case 2: leader has XTerm:
		//nextIndex = leader's last entry for XTerm
		//Case 3: follower's log is too short:
		//nextIndex = XLen
		if reply.XTerm == -1 {
			if rf.lastIncludedIndex >= reply.XLen {
				go rf.handleInstallSnapshot(server)
				return
			}
			rf.nextIndex[server] = reply.XLen
			DPrintf("%v 1rf.nextIndex[server] %v", rf.me, rf.nextIndex[server])
		} else {

			i := rf.nextIndex[server] - 1
			if i < rf.lastIncludedIndex {
				i = rf.lastIncludedIndex
			}
			for i > rf.lastIncludedIndex && rf.log[i-rf.lastIncludedIndex].Term > reply.XTerm {
				i--
			}
			if i == rf.lastIncludedIndex && reply.XTerm != rf.log[i-rf.lastIncludedIndex].Term {
				go rf.handleInstallSnapshot(server)
				return
			}
			if rf.log[i-rf.lastIncludedIndex].Term == reply.XTerm {
				rf.nextIndex[server] = i + 1
				DPrintf("%v 2rf.nextIndex[server] %v", rf.me, rf.nextIndex[server])
			} else {
				if reply.XIndex <= rf.lastIncludedIndex {
					go rf.handleInstallSnapshot(server)
					return
				}

				//Xindex may back up to 0!
				rf.nextIndex[server] = reply.XIndex
				DPrintf("%v 3rf.nextIndex[server] %v", rf.me, rf.nextIndex[server])
			}
		}
	}
}

func (rf *Raft) SendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// Receiver implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. Reply false if log doesn’t contain an entry at prevLogIndex
// whose term matches prevLogTerm (§5.3)
// 3. If an existing entry conflicts with a new one (same index
// but different terms), delete the existing entry and all that
// follow it (§5.3)
// 4. Append any new entries not already in the log
// 5. If leaderCommit > commitIndex, set commitIndex =
// min(leaderCommit, index of last new entry)

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//this section timeout
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	rf.timestamp = time.Now()
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		rf.state = follower
		rf.persist()
	}
	//bug2 heartbeat will also help update term!
	if len(args.Entries) == 0 {
		DPrintf("AppendEntries: receive heartbeat from %v\n", args.LeaderId)
	} else {
		DPrintf("AppendEntries: server %v get leader %v appendEntries! %+v\n", rf.me, args.LeaderId, args)
	}
	//outdated rpc
	if args.PrevLogIndex < rf.lastIncludedIndex {
		reply.Success = true
		reply.Term = rf.currentTerm
		return
	} else if len(rf.log)+rf.lastIncludedIndex <= args.PrevLogIndex || rf.log[args.PrevLogIndex-rf.lastIncludedIndex].Term != args.PrevLogTerm {
		//DPrintf("AppendEntries failed")
		reply.Term = rf.currentTerm
		reply.Success = false
		if len(rf.log)+rf.lastIncludedIndex <= args.PrevLogIndex {
			reply.XTerm = -1
			reply.XLen = len(rf.log) + rf.lastIncludedIndex
		} else {
			reply.XTerm = rf.log[args.PrevLogIndex-rf.lastIncludedIndex].Term
			k := args.PrevLogIndex
			//fuck bug! this is reply.XTerm! not reply.Term!!
			for k > rf.lastIncludedIndex && rf.log[k-rf.lastIncludedIndex].Term == reply.XTerm {
				k--
			}
			reply.XIndex = k + 1
		}
		return
	}
	//bug, this may cause the array out of range!
	//for i := 0; i < len(args.Entries); i++ {
	//	index := (args.PrevLogIndex - rf.lastIncludedIndex) + 1 + i
	//	if index < len(rf.log) && rf.log[index].Term != args.Entries[i].Term {
	//		rf.log = rf.log[:index]
	//		rf.log = append(rf.log, args.Entries[i:]...)
	//		break
	//	} else if index == len(rf.log) {
	//		rf.log = append(rf.log, args.Entries[i:]...)
	//		break
	//	}
	//}

	//directly ignore the conflict and delete the all logs, then append
	//if len(args.Entries) > 0 && len(rf.log)+rf.lastIncludedIndex > args.PrevLogIndex+1 {
	//	rf.log = rf.log[:args.PrevLogIndex+1-rf.lastIncludedIndex]
	//}
	//rf.log = append(rf.log, args.Entries...)
	for idx, log := range args.Entries {
		ridx := args.PrevLogIndex - rf.lastIncludedIndex + 1 + idx
		if ridx < len(rf.log) && rf.log[ridx].Term != log.Term {
			rf.log = rf.log[:ridx]
			rf.log = append(rf.log, args.Entries[idx:]...)
			break
		} else if ridx == len(rf.log) {
			rf.log = append(rf.log, args.Entries[idx:]...)
			break
		}
	}

	rf.persist()

	reply.Term = rf.currentTerm
	reply.Success = true
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, len(rf.log)+rf.lastIncludedIndex-1)
		rf.cond.Signal()
	}
	return
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.state == leader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	e.Encode(rf.lastIncludedIndex)
	e.Encode(rf.lastIncludedTerm)

	raftstate := w.Bytes()
	rf.persister.Save(raftstate, rf.snapshot)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) == 0 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []Entry
	var lastIncludedIndex int
	var lastIncludedTerm int
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil ||
		d.Decode(&lastIncludedIndex) != nil ||
		d.Decode(&lastIncludedTerm) != nil {
		DPrintf("readPersist readstate failed\n")
	} else {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
		rf.lastIncludedIndex = lastIncludedIndex
		rf.lastIncludedTerm = lastIncludedTerm

		rf.commitIndex = lastIncludedIndex
		rf.lastApplied = lastIncludedIndex
		return
	}
}

func (rf *Raft) readSnapshot(data []byte) {
	if data == nil || len(data) == 0 { // bootstrap without any state?
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.snapshot = data
	return
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//include not commit log, so reject
	if index > rf.commitIndex || index <= rf.lastIncludedIndex {
		DPrintf("Snapshot: index range error")
		return
	}
	//include 9, index =11, so need to 10 ,11, log start from 12
	rf.snapshot = snapshot
	//the log[0] is always the nil!
	rf.lastIncludedTerm = rf.log[index-rf.lastIncludedIndex].Term
	rf.log = rf.log[index-rf.lastIncludedIndex:]
	rf.lastIncludedIndex = index
	if rf.lastApplied < index {
		rf.lastApplied = index
	}
	rf.persist()
}

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

type InstallSnapshotReply struct {
	Term int
}

//Receiver implementation:
//1. Reply immediately if term < currentTerm
//2. Create new snapshot file if first chunk (offset is 0)
//3. Write data into snapshot file at given offset
//4. Reply and wait for more data chunks if done is false
//5. Save snapshot file, discard any existing or partial snapshot
//with a smaller index
//6. If existing log entry has same index and term as snapshot’s
//last included entry, retain log entries following it and reply
//7. Discard the entire log
//8. Reset state machine using snapshot contents (and load
//snapshot’s cluster configuration)

// for server just install snapshot
func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//if currentTerm > Term, ignore! (it maybe the old InstallSnapshot
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		rf.timestamp = time.Now()
		return
	}
	if args.Term > rf.currentTerm {
		//fuck bug, args.Term write to reply.Term
		rf.currentTerm = args.Term
		rf.timestamp = time.Now()
		rf.votedFor = -1
		rf.state = follower
		// may have the bug
		return
	}

	flag := false
	//If existing log entry has same index and term as snapshot’s
	//last included entry, retain log entries following it and reply
	for i := 0; i < len(rf.log); i++ {
		if i+rf.lastIncludedIndex == args.LastIncludedIndex && rf.log[i].Term == args.LastIncludedTerm {
			rf.log = rf.log[i:]
			flag = true
			break
		}
	}
	if !flag {
		//Discard the entire log
		rf.log = make([]Entry, 0)
		rf.log = append(rf.log, Entry{Term: rf.lastIncludedTerm, Cmd: nil})
	}

	//Reset state machine using snapshot contents
	rf.snapshot = args.Data
	rf.lastIncludedIndex = args.LastIncludedIndex
	rf.lastIncludedTerm = args.LastIncludedTerm

	if rf.commitIndex < args.LastIncludedIndex {
		rf.commitIndex = args.LastIncludedIndex
	}
	if rf.lastApplied < args.LastIncludedIndex {
		rf.lastApplied = args.LastIncludedIndex
	}

	reply.Term = rf.currentTerm

	msg := ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Data,
		SnapshotIndex: args.LastIncludedIndex,
		SnapshotTerm:  args.LastIncludedTerm,
	}

	rf.applyCh <- msg
	rf.persist()
}

func (rf *Raft) handleInstallSnapshot(server int) {
	rf.mu.Lock()
	if rf.state != leader {
		rf.mu.Unlock()
		return
	}
	reply := &InstallSnapshotReply{}
	args := &InstallSnapshotArgs{
		Term:              rf.currentTerm,
		LeaderId:          rf.me,
		LastIncludedIndex: rf.lastIncludedIndex,
		LastIncludedTerm:  rf.lastIncludedTerm,
		Data:              rf.snapshot,
	}
	rf.mu.Unlock()
	ok := rf.sendInstallSnapshot(server, args, reply)
	if !ok {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.state = follower
		rf.votedFor = -1
		rf.timestamp = time.Now()
		rf.persist()
		return
	}
	rf.nextIndex[server] = rf.lastIncludedIndex + 1
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int
	CandidateId  int
	LastlogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Entry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	XTerm   int // term in the conflicting entry (if any)
	XIndex  int // index of first entry with that term (if any)
	XLen    int // log length
}

//Receiver implementation:
//1. Reply false if term < currentTerm (§5.1)
//2. If votedFor is null or candidateId, and candidate’s log is at
//least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	//if args.Term == rf.currentTerm, it may elect other candidates, so dont vote
	if args.Term > rf.currentTerm {
		rf.votedFor = -1
		rf.currentTerm = args.Term
		rf.state = follower
		//changed voteFor
		rf.persist()
	}

	lastLogIndex := len(rf.log) - 1
	lastLogTerm := rf.log[lastLogIndex].Term

	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && (args.LastLogTerm > lastLogTerm || args.LastLogTerm == lastLogTerm && args.LastlogIndex >= lastLogIndex+rf.lastIncludedIndex) {
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateId
		rf.state = follower
		rf.timestamp = time.Now()
		rf.persist()
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
	} else {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
	}
	return
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != leader || rf.killed() {
		return -1, -1, false
	}
	rf.log = append(rf.log, Entry{Term: rf.currentTerm, Cmd: command})
	rf.persist()

	defer rf.heartTimer.Reset(time.Duration(15) * time.Millisecond)

	return len(rf.log) - 1 + rf.lastIncludedIndex, rf.currentTerm, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	rd := rand.New(rand.NewSource(int64(rf.me)))
	for !rf.killed() {
		// Your code here (3A)
		// Check if a leader election should be started.
		rf.mu.Lock()

		if rf.state != leader && time.Since(rf.timestamp) > time.Duration(ElectionTimeout+rd.Intn(150))*time.Millisecond {
			DPrintf("heartbeat timeout, start elect")
			go rf.elect()
		}

		rf.mu.Unlock()
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

//On conversion to candidate, start election:
//• Increment currentTerm
//• Vote for self
//• Reset election timer
//• Send RequestVote RPCs to all other servers
//• If votes received from majority of servers: become leader
//• If AppendEntries RPC received from new leader: convert to follower
//• If election timeout elapses: start new election

func (rf *Raft) elect() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm++
	rf.state = candidate
	rf.votedFor = rf.me
	rf.timestamp = time.Now()
	rf.persist()
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastlogIndex: len(rf.log) - 1 + rf.lastIncludedIndex,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}

	rf.muVote.Lock()
	rf.voteCnt = 1
	rf.muVote.Unlock()
	//voteCnt1 := 1
	//var muVote1 sync.Mutex

	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		//go rf.handleRequestVote(i, &args, &voteCnt1, &muVote1)
		go rf.handleRequestVote(i, &args)

	}
}

//If commitIndex > lastApplied: increment lastApplied, apply log[lastApplied] to state machine (§5.3)
//• If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)

func (rf *Raft) check() {
	for !rf.killed() {

		rf.mu.Lock()
		for rf.commitIndex <= rf.lastApplied {
			rf.cond.Wait()
		}
		msgs := make([]ApplyMsg, 0, rf.commitIndex-rf.lastApplied)
		oldLastApplied := rf.lastApplied
		for rf.commitIndex > oldLastApplied {
			oldLastApplied += 1
			if oldLastApplied <= rf.lastIncludedIndex {
				continue
			}

			msg := ApplyMsg{
				CommandValid: true,
				Command:      rf.log[oldLastApplied-rf.lastIncludedIndex].Cmd,
				CommandIndex: oldLastApplied,
			}
			msgs = append(msgs, msg)
		}
		rf.mu.Unlock()
		//for _, msg := range msgs {
		//	rf.applyCh <- msg
		//}
		for _, m := range msgs {
			rf.mu.Lock()
			if m.CommandIndex != rf.lastApplied+1 {
				rf.mu.Unlock()
				continue
			}
			rf.mu.Unlock()
			rf.applyCh <- m
			rf.mu.Lock()
			if m.CommandIndex != rf.lastApplied+1 {
				rf.mu.Unlock()
				continue
			}
			rf.lastApplied = m.CommandIndex
			rf.mu.Unlock()
		}

		//time.Sleep(10 * time.Millisecond)
	}
}

func (rf *Raft) handleRequestVote(server int, args *RequestVoteArgs) {
	//func (rf *Raft) handleRequestVote(server int, args *RequestVoteArgs, voteCnt1 *int, muVote1 *sync.Mutex) {
	reply := &RequestVoteReply{}
	ok := rf.sendRequestVote(server, args, reply)
	if !ok {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term != rf.currentTerm {
		return
	}

	//if not the largest Term, start a new election
	if reply.Term > rf.currentTerm {
		rf.state = follower
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		rf.persist()
		return
	}

	if reply.VoteGranted {
		rf.muVote.Lock()
		rf.voteCnt++
		//muVote1.Lock()
		//*voteCnt1++
		//DPrintf("vote cnt is %v", rf.voteCnt)
		//if *voteCnt1 > len(rf.peers)/2 && rf.state == candidate {
		if rf.voteCnt > len(rf.peers)/2 && rf.state == candidate {
			rf.state = leader
			for j := 0; j < len(rf.peers); j++ {
				rf.nextIndex[j] = len(rf.log) + rf.lastIncludedIndex
				rf.matchIndex[j] = rf.lastIncludedIndex
			}
			go rf.SendHeartbeat()
		}
		//muVote1.Unlock()
		rf.muVote.Unlock()
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]Entry, 0)
	rf.log = append(rf.log, Entry{Term: 0, Cmd: nil})
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	rf.applyCh = applyCh
	rf.timestamp = time.Now()
	rf.state = follower
	rf.voteCnt = 0
	rf.cond = sync.NewCond(&rf.mu)
	rf.heartTimer = time.NewTimer(0)

	// initialize from state persisted before a crash
	rf.readSnapshot(persister.ReadSnapshot())
	rf.readPersist(persister.ReadRaftState())
	//first recover log,currentTerm,votedFor
	for i := 0; i < len(rf.peers); i++ {
		//initialized to leader last log index + 1
		rf.nextIndex[i] = len(rf.log) + rf.lastIncludedIndex
		//initialized to 0, increases monotonically
		rf.matchIndex[i] = rf.lastIncludedIndex
	}

	// start ticker goroutine to start elections
	go rf.ticker()

	go rf.check()
	return rf
}
