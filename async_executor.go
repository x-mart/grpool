package grpool

// 异步任务执行器, 无阻塞, 不关心执行结果.
// 协程安全.
type AsyncExecutor struct {
	p *Pool
}

func NewAsyncExecutor(p *Pool) *AsyncExecutor {
	return &AsyncExecutor{
		p: p,
	}
}

// 添加任务. 会立刻交给 pool 执行
func (e *AsyncExecutor) AddTask(task AsyncFunc) bool {
	return e.p.AddTask(&task)
}
