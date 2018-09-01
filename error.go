package grpool

// 包装 FutureTask 的错误
type FutureTaskError struct {
	Msg string
}

func NewFutureTaskError(msg string) *FutureTaskError {
	return &FutureTaskError{
		Msg: msg,
	}
}

func (e *FutureTaskError) Error() string {
	return e.Msg
}

// 包装 FutureFunc 返回的错误
type FutureFuncError struct {
	Err error
}

func NewFutureFuncError(err error) *FutureFuncError {
	return &FutureFuncError{
		Err: err,
	}
}

func (e *FutureFuncError) Error() string {
	return e.Err.Error()
}
