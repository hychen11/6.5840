package kvraft

const (
	OK                = "OK"
	ErrNoKey          = "ErrNoKey"
	ErrWrongLeader    = "ErrWrongLeader"
	ErrTimeout        = "ErrTimeout"
	ErrCommitFailed   = "ErrCommitFailed"
	ErrChanClose      = "ErrChanClose"
	ErrOldRequest     = "ErrOldRequest"
	ErrLeaderOutDated = "LeaderOutDated"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientID  int64
	RequestID int64
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	ClientID  int64
	RequestID int64
}

type GetReply struct {
	Err   Err
	Value string
}
