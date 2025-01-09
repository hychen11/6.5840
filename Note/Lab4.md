# 4A

bug：

commit是在appendEntries里查看是否majority已经提交，是在sendHeartbeat后进行的

但是这里要求

`// Check that ops are committed fast enough, better than 1 per heartbeat interval`

所以也就是每次start时都要提交一个heartbeat，所以使用chan比time.Sleep更好

```go
type Raft struct{
		heartTimer *time.Timer
}

for !rf.killed(){
  	<-rf.heartTimer.C
  	//......
  	rf.heartTimer.Reset(time.Duration(101)*time.Millisecond)
}


rf.heartTimer=time.NewTimer(0)
```



上一轮的选举迟到的RPC会影响下一轮的选举





kv server submit a request -> `kv.rf.start(cmd)`

然后raft会进行投票和返回是否成功commit的结果

这里通过channel返回结果

```go
ch:=kv.commitChan[index]
defer func() {
  kv.mu.Lock()
  close(ch)
  delete(kv.commitChan, index)
  kv.mu.Unlock()
}()
select{
case <-time.After(TimeOut):
		reply.Err=ErrTimeOut
case result<-ch:
		reply=result
}
```

**Raft 保证** startIndex **唯一**：每次调用 Start 方法返回的 startIndex 是递增且唯一的，对当前 Leader 是全局唯一。

**日志覆盖不影响索引唯一性**：即使日志覆盖，新 Leader 为重复请求分配的新索引仍然唯一。

**使用** startIndex **作为键的好处**：避免覆盖管道，确保每个请求有独立的等待通道。

这里一个kv server就是一个raft里的一个peer

然后如果当前kv server是leader时，就需要发送状态
