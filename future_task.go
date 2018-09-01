package grpool

import (
	"fmt"
	"time"
	"sync/atomic"
)

// FutureFunc, error != nil 时将会丢弃 interface{}
type FutureFunc func() (interface{}, error)

type FutureTask struct {
	f          FutureFunc
	resultChan chan interface{}
	aborted    uint32
}

func NewFutureTask(f FutureFunc) *FutureTask {
	return &FutureTask{
		f:          f,
		resultChan: make(chan interface{}, 1),
		aborted:    0,
	}
}

func (t *FutureTask) Run() {
	defer func() {
		if err := recover(); err != nil {
			t.Abort(fmt.Sprintf("task.Run panic: %v", err))
		}
	}()

	if t.IsAborted() {
		return
	}

	result, err := t.f()
	if err != nil {
		t.tReturn(NewFutureFuncError(err))
	} else {
		t.tReturn(result)
	}
}

func (t *FutureTask) PoolCloseCallBack() {
	t.Abort("pool closed")
}

// 终止任务. 首次调用有效.
// 未执行的任务将不会执行. 正在执行的任务将会丢弃结果. 已经执行完成的任务不受影响.
// 返回 tReturn FutureTaskError, 已经执行的结果不受影响
func (t *FutureTask) Abort(msg string) {
	if t.IsAborted() { // 加这个性能稍有提升
		return
	}
	if !atomic.CompareAndSwapUint32(&t.aborted, 0, 1) {
		return
	}
	t.tReturn(NewFutureTaskError(msg))
}

// 是否终止
func (t *FutureTask) IsAborted() bool {
	return atomic.LoadUint32(&t.aborted) == 1
}

// 获取结果, 会阻塞.
//
// @return
//     interface{} 为 FutureFunc 返回的 interface{},
//     error 为 Task 执行的错误(FutureTaskError) 或者 FutureFunc 返回的错误(FutureFuncError).
func (t *FutureTask) GetResult() (interface{}, error) {
	select {
	case result, ok := <-t.resultChan:
		if !ok {
			return nil, NewFutureTaskError("result chan closed")
		}
		return t.pickResult(result)
	}
}

// 获取结果, 可以设置超时. 参数 timer 可以用于限制一组 task 的超时时间
//
// @return
//     interface{} 为 FutureFunc 返回的 interface{}.
//     bool 是否超时, true-超时, false-未超时.
//     error 为 Task 执行的错误(FutureTaskError) 或者 FutureFunc 返回的错误(FutureFuncError).
func (t *FutureTask) GetResultWithTimeout(timer *time.Timer) (interface{}, bool, error) {
	select {
	case <-timer.C:
		return nil, true, NewFutureTaskError("time out here")

	case result, ok := <-t.resultChan:
		if !ok {
			return nil, false, NewFutureTaskError("result chan closed")
		}
		result, err := t.pickResult(result)
		return result, false, err
	}
}

// 获取结果, 非阻塞. 立刻返回, 无论是否得到结果
//
// @return
//     interface{} 为 FutureFunc 返回的 interface{}.
//     error 为 Task 执行的错误(FutureTaskError) 或者 FutureFunc 返回的错误(FutureFuncError).
func (t *FutureTask) GetResultNoWait() (interface{}, error) {
	select {
	case result, ok := <-t.resultChan:
		if !ok {
			return nil, NewFutureTaskError("result chan closed")
		}
		return t.pickResult(result)

	default:
		return nil, NewFutureTaskError("no result yet")
	}
}

// 返回结果. 成功返回 true, 失败返回 false. 如果已经有结果了会 false
func (t *FutureTask) tReturn(result interface{}) bool {
	select {
	case t.resultChan <- result:
		return true
	default:
		return false
	}
}

// 分离 resultChan 中的正常结果和 error
func (t *FutureTask) pickResult(result interface{}) (interface{}, error) {
	if err, ok := result.(*FutureTaskError); ok {
		return nil, err
	}
	if err, ok := result.(*FutureFuncError); ok {
		return nil, err
	}
	return result, nil
}
