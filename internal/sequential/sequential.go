package sequential

import (
	"errors"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/arcanericky/sleepsort/internal/sleepsort"
	"github.com/arcanericky/sleepsort/internal/streamedresult"
)

var ErrCancelled error = errors.New("cancelled")

// SequentialSleepSort is a class that will sequentially sleep sort a
// list of uints
type SequentialSleepSort struct{}

func (sss SequentialSleepSort) StreamSort(list []uint) chan streamedresult.Result {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	resultChan := make(chan streamedresult.Result, 1)

	go func() {
		startTime := time.Now()
		previousResult, currentResult, err := initialRound(list, sigChan)
		resultChan <- streamedresult.New(1, startTime, time.Now(), previousResult, currentResult, err)
		if err != nil {
			close(resultChan)
			return
		}

		var round uint = 1
		var multiplier uint = 1
		for !reflect.DeepEqual(previousResult, currentResult) {
			copy(previousResult, currentResult)
			round++
			multiplier *= 2
			startTime = time.Now()
			currentResult, err = nextRound(round, multiplier, previousResult, sigChan)
			resultChan <- streamedresult.New(int(round), startTime, time.Now(), previousResult, currentResult, err)
			if err != nil {
				close(resultChan)
				return
			}
		}

		close(resultChan)
	}()

	return resultChan
}

func launch(list []uint, round uint, multiplier uint) (chan []uint, chan struct{}) {
	return (sleepsort.SleepSort{Multiplier: multiplier}).Sort(list)
}

func initialRound(list []uint, sigChan chan os.Signal) ([]uint, []uint, error) {
	var newResult []uint

	result := make([]uint, len(list))
	copy(result, list)

	newResultChan, newStopChan := launch(result, 1, 1)
	resultChan, stopChan := launch(result, 1, 1)
	for i := 0; i < 2; i++ {
		select {
		case newResult = <-newResultChan:
		case result = <-resultChan:
		case <-sigChan:
			signal.Stop(sigChan)
			close(newStopChan)
			close(stopChan)
			<-newResultChan
			<-resultChan
			return nil, nil, ErrCancelled
		}
	}

	return result, newResult, nil
}

func nextRound(round uint, multiplier uint, list []uint, sigChan chan os.Signal) ([]uint, error) {
	var result []uint
	resultChan, stopChan := launch(list, round, multiplier)
	select {
	case result = <-resultChan:
	case <-sigChan:
		signal.Stop(sigChan)
		close(stopChan)
		<-resultChan
		return nil, ErrCancelled
	}

	return result, nil
}
