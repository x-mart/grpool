package grpool

import (
	"sync/atomic"
	"github.com/sirupsen/logrus"
)

type Task interface {
	Run()
	// pool 关闭时回调未执行的任务
	PoolCloseCallBack()
}

// 协程池. 协程安全
// 协程池中包含 1 个 master 和 0 ~ maxGoroutineNum 个 worker.
// master 负责获取任务、执行任务和创建 worker.
// worker 负责获取任务和执行任务. worker 一旦创建, 会和 master 公平竞争任务. worker 只有第一个任务由 master 分配.
// worker 竞争不到任务时, 会结束自己. 因为 timer 的成本大于新创建一个协程, 这里没有用空闲时间.
// 限制 worker 数量通过获取和归还令牌来实现.
type Pool struct {
	taskChan        chan Task
	workerSemaChan  chan struct{}
	maxGoroutineNum int

	closed uint32 // 使用 closed + atomic 来同步
}

// 创建协程池.
//     @param maxBuffer 协程池的最大缓冲, 待执行任务数量超过该值时添加任务失败
//     @param maxGoroutineNum 协程池的最大 goroutine 数量, 该值最小为 1
func NewPool(maxBuffer int, maxGoroutineNum int) *Pool {
	p := &Pool{
		maxGoroutineNum: maxGoroutineNum,
		closed:          0,
	}

	p.taskChan = make(chan Task, maxBuffer)
	p.workerSemaChan = make(chan struct{}, maxGoroutineNum-1)

	go p.master() // master goroutine

	return p
}

// 添加协程.
// 协程池中缓冲的任务超过 maxBuffer 时返回 false.
// 协程池已关闭会 panic.
func (p *Pool) AddTask(task Task) bool {
	select {
	case p.taskChan <- task:
		return true
	default:
		return false
	}
}

// 关闭 pool. 协程安全
func (p *Pool) Close() {
	if p.IsClosed() {
		return
	}
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return
	}

	close(p.taskChan)

	// 消费完剩余的所有 task, 避免有进程阻塞
	removed := 0
LOOP:
	for {
		select {
		case task, ok := <-p.taskChan:
			if !ok {
				break LOOP
			}
			task.PoolCloseCallBack()
			removed += 1
		default:
			break LOOP
		}
	}
	logrus.Infof("close pool. removed %v task(s)", removed)
}

// pool 是否关闭. 因为协程池的状态只能由 创建 -> 关闭. 关闭状态为 true 时可靠
func (p Pool) IsClosed() bool {
	return atomic.LoadUint32(&p.closed) == 1
}

// master 协程
func (p *Pool) master() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("master panic: %v", err)
		}
	}()

	// 退出 master 时关闭 pool
	defer p.Close()

LOOP:
	for {
		// 避免竞争 close pool 后剩下的任务
		if p.IsClosed() {
			break LOOP
		}

		select {
		case task, ok := <-p.taskChan:
			if !ok {
				logrus.Infof("pool closed. stop master")
				break LOOP
			}

			// 队列中无任务时, 由 master 处理
			if len(p.taskChan) == 0 {
				task.Run()
			} else {
				// 否则, 分配给 worker 处理
				ret := p.dispatchWorker(task)

				// worker 繁忙时, 由 master 处理
				if !ret {
					task.Run()
				}
			}
		}
	}
}

// 分配任务到 worker
func (p *Pool) dispatchWorker(task Task) bool {
	// 获取令牌
	if !p.getWorkerSema() {
		return false
	}

	// 创建 worker
	go p.worker(task)

	return true
}

// worker 协程
func (p *Pool) worker(task Task) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("worker panic: %v", err)
		}
	}()

	// 结束时归还令牌
	defer p.returnWorkerSema()

	// 处理 task
	task.Run()

	// 处理完成后不立刻退出, 继续监听处理
LOOP:
	for {
		// 避免竞争 close pool 后剩下的任务
		if p.IsClosed() {
			break LOOP
		}

		select {
		case task, ok := <-p.taskChan:
			if !ok {
				logrus.Infof("pool closed. stop worker")
				break LOOP
			}

			task.Run()

		default:
			// 因为用 time.NewTimer 设置超时的成本大于新创建一个协程
			// 这里竞争不到 task 就直接退出
			break LOOP
		}
	}
}

// 获取 worker 令牌
func (p *Pool) getWorkerSema() bool {
	select {
	case p.workerSemaChan <- struct{}{}:
		return true
	default:
		return false
	}
}

// 归还 worker 令牌
func (p *Pool) returnWorkerSema() bool {
	select {
	case <-p.workerSemaChan:
		return true
	default:
		return false
	}
}
