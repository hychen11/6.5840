# 3A

一些问题

```go
mu.Lock()
defer mu.Unlock()
go funcA()
```

这里的mu在go funcA前不会Unlock！



# 3B

slice A add to slice B

```go
B=append(B,A...)
```

Condition

```go
mu 		sync.Mutex
cond 	*sync.Cond
//Initialize
cond=sync.NewCond(&mu)
```

