package streamedresult

import "time"

type Result struct {
	Round          uint
	BeginTime      time.Time
	EndTime        time.Time
	PreviousResult []uint
	CurrentResult  []uint
	Err            error
}

func New(round int, begin, end time.Time, previous, current []uint, err error) Result {
	r := Result{
		Round:          uint(round),
		BeginTime:      begin,
		EndTime:        end,
		PreviousResult: make([]uint, len(previous)),
		CurrentResult:  make([]uint, len(current)),
		Err:            err,
	}

	copy(r.PreviousResult, previous)
	copy(r.CurrentResult, current)

	return r
}
