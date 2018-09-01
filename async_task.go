package grpool

import (
	"github.com/sirupsen/logrus"
)

type AsyncFunc func()

func NewAsyncTask(f AsyncFunc) *AsyncFunc {
	return &f
}

func (t *AsyncFunc) Run() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("task.Run panic: %v", err)
		}
	}()

	(*t)()
}

func (t *AsyncFunc) PoolCloseCallBack() {
	logrus.Warnf("pool closed and GGWP")
}
