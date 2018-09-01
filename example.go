package grpool

import (
	"fmt"
	"time"
	"math/rand"
)

// example 1 打印日志的 async 任务
var random = rand.New(rand.NewSource(time.Now().Unix()))
var LogPool = NewPool(1000, 5)

func SendLog(msg string) error {
	time.Sleep(time.Duration(random.Intn(20)+20) * time.Duration(time.Millisecond))
	return nil
}

func SendLogs(msgs []string) {
	executor := NewAsyncExecutor(LogPool)
	for i, msg := range msgs {
		index := i
		message := msg
		executor.AddTask(func() {
			SendLog(message)
			fmt.Printf("[Task %v] send log finish. msg: %v\n", index, message)
		})
	}
}

func TestSendLog() {
	var msgs []string
	msgNum := 20
	for i := 0; i < msgNum; i++ {
		msgs = append(msgs, fmt.Sprintf("i am msg %v", i))
	}

	SendLogs(msgs)
}

// example 2 查询 mysql Future 任务
var MysqlPool = NewPool(1000, 5)

type User struct {
	ID   int
	Name string
}

func SelectUser(id int) (*User, error) {
	cost := random.Intn(100)
	time.Sleep(time.Duration(cost) * time.Duration(time.Millisecond))

	if cost < 2 {
		panic("connection refused")
	}
	if cost > 98 {
		return nil, fmt.Errorf("time out: %vms", cost)
	}

	return &User{
		ID:   id,
		Name: fmt.Sprintf("I AM %v", id),
	}, nil
}

func BatchSelectUser(ids []int) map[int]*User {
	executor := NewFutureExecutor(MysqlPool)

	for _, id := range ids {
		lId := id
		executor.AddTask(func() (interface{}, error) {
			return SelectUser(lId)
		})
	}

	resultList, errorList := executor.WaitWithTimeout(1000 * time.Duration(time.Millisecond))

	userMap := make(map[int]*User, 0)
	for i, id := range ids {
		if errorList[i] != nil {
			fmt.Printf("[ERROR] select user error. id: %v, err: %v\n", id, errorList[i])
		} else {
			user, ok := resultList[i].(*User)
			if !ok { // impossible
				fmt.Printf("[ERROR] unknown result. id: %v\n", id)
			} else {
				fmt.Printf("[INFO] select user. id: %v, user: %v\n", id, user)
				userMap[id] = user
			}
		}
	}

	return userMap
}

func TestBatchSelectUser() {
	var ids []int
	userNum := 100
	for i := 0; i < userNum; i++ {
		ids = append(ids, 10000+i)
	}

	users := BatchSelectUser(ids)

	fmt.Println(users[10005])
	fmt.Println(users[10020])

}
