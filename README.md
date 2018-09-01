# Introduction
Goroutine pool, async executor and future executor. features 
- Setting max waiting task buffer and max goroutine num
- Reusing goroutine and resizing goroutine num dynamically
- Using pool by multi-goroutine safely and sharing pool between multi-executor
- Getting task result and setting tasks timeout

# Examples
AsyncExecutor handle async tasks that you don't care their result and you no need to wait them finish.
```Go

// new pool with max task buffer and max goroutine num. Pool can shared between multi-executor.
p := grpool.NewPool(1000, 5)

// create a async executor
e := grpool.NewAsyncExecutor(p)

// add some tasks, and they will be executed async
for i := 0; i < 10; i++ {
    e.AddTask(func() {
        // do something
    })
}

// do other thing
```

FutureExecutor handle future task. You can get the task result and set several task timeout.
```Go

// new pool with max task buffer and max goroutine num. pool can shared between multi-executor.
p := grpool.NewPool(1000, 5)

// create a future executor
e := grpool.NewFutureExecutor(p)

// add some tasks
for i := 0; i < 10; i++ {
    e.AddTask(func() (interface{}, error) {
        // do something
        return result, err
    })
}

// wait tasks finished or timeout
resultList, errorList := e.WaitWithTimeout(time.Second)

// handle task result
```

