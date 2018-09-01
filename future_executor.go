package grpool

import "time"

// Future 任务执行器, 可返回执行结果.
// 非协程安全.
type FutureExecutor struct {
	p *Pool

	tasks []*FutureTask
}

func NewFutureExecutor(p *Pool) *FutureExecutor {
	return &FutureExecutor{
		p:     p,
		tasks: make([]*FutureTask, 0),
	}
}

// 添加任务
func (e *FutureExecutor) AddTask(f FutureFunc) {
	task := NewFutureTask(f)
	e.tasks = append(e.tasks, task)

	ok := e.p.AddTask(task)
	if !ok {
		task.Abort("pool is full")
	}
}

// 获取结果.
// @return
//     []interface{} 为 FutureFunc 返回的 interface{}, 顺序和 AddTask 顺序完全一致
//     []error 包括 Task 执行的错误(FutureTaskError)以及 FutureFunc 返回的错误(FutureFuncError), 顺序和 AddTask 顺序完全一致
func (e *FutureExecutor) Wait() ([]interface{}, []error) {
	taskNum := len(e.tasks)
	resultList := make([]interface{}, taskNum)
	errorList := make([]error, taskNum)
	for index, task := range e.tasks {
		result, err := task.GetResult()
		if err != nil {
			errorList[index] = err
		} else {
			resultList[index] = result
		}
	}
	return resultList, errorList
}

// 获取结果.
// 超时后, 未开始执行的任务将会被取消, 未完成执行的任务结果将被丢弃. 已经执行完成的任务结果正常返回
// @return
//     []interface{} 为 FutureFunc 返回的 interface{}, 顺序和 AddTask 顺序完全一致
//     []error 包括 Task 执行的错误(FutureTaskError)以及 FutureFunc 返回的错误(FutureFuncError), 顺序和 AddTask 顺序完全一致
func (e *FutureExecutor) WaitWithTimeout(d time.Duration) ([]interface{}, []error) {
	taskNum := len(e.tasks)
	resultList := make([]interface{}, taskNum)
	errorList := make([]error, taskNum)
	timer := time.NewTimer(d)
	hasTimeout := false
	for index, task := range e.tasks {
		if !hasTimeout {
			result, timeout, err := task.GetResultWithTimeout(timer)
			if err != nil {
				errorList[index] = err
			} else {
				resultList[index] = result
			}

			if timeout {
				hasTimeout = true

				// 发生超时, 终止剩余的、未执行的任务
				taskNum := len(e.tasks)
				for i := index + 1; i < taskNum; i++ {
					e.tasks[i].Abort("time out abort")
				}
			}
		} else { // 超时后, 无等待结果获取
			result, err := task.GetResultNoWait()
			if err != nil {
				errorList[index] = err
			} else {
				resultList[index] = result
			}
		}
	}
	return resultList, errorList
}
