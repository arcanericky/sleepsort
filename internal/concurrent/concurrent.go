package concurrent

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

type chanPair struct {
	sort chan []uint
	stop chan struct{}
}

// ConcurrentSleepSort is a class that will concurrently sleep sort a
// list of uints
type ConcurrentSleepSort struct {
	MaxRounds uint
}

func (css ConcurrentSleepSort) launchSorters(list []uint) (chan []uint, chan struct{}) {
	outChan := make(chan []uint, 1)
	sc := make(chan struct{}, 1)
	var multiplier uint = 1

	var cp []chanPair

	sortChan, stopChan := (sleepsort.SleepSort{Multiplier: 1}).Sort(list)
	cp = append(cp, chanPair{sort: sortChan, stop: stopChan})

	for i := uint(1); i < css.MaxRounds+1; i++ {
		sortChan, stopChan = (sleepsort.SleepSort{Multiplier: multiplier}).Sort(list)
		cp = append(cp, chanPair{sort: sortChan, stop: stopChan})
		multiplier *= 2
	}

	go func() {
		for _, c := range cp {
			select {
			case <-sc:
				close(c.stop)
				<-c.sort
			case data := <-c.sort:
				outChan <- data
			}
		}
		close(outChan)
	}()

	return outChan, sc
}

func (css ConcurrentSleepSort) streamResults(sortChan chan []uint, stopChan chan struct{}) chan streamedresult.Result {
	var previousResult []uint
	resultChan := make(chan streamedresult.Result, 1)
	go func() {
		for i := 1; ; i++ {
			startTime := time.Now()
			if i == 1 {
				previousResult = <-sortChan
			}

			var currentResult []uint
			var ok bool
			currentResult, ok = <-sortChan
			endTime := time.Now()
			if !ok {
				resultChan <- streamedresult.New(i, startTime, endTime, previousResult, currentResult, errors.New("sort failed"))
				close(resultChan)
				break
			}

			if reflect.DeepEqual(previousResult, currentResult) {
				resultChan <- streamedresult.New(i, startTime, endTime, previousResult, currentResult, nil)
				close(stopChan)
				<-sortChan
				close(resultChan)
				break
			}

			resultChan <- streamedresult.New(i, startTime, endTime, previousResult, currentResult, nil)
			copy(previousResult, currentResult)
		}
	}()

	return resultChan
}

// Sort will concurrently sleep sort a list of uints
func (css ConcurrentSleepSort) StreamSort(list []uint) chan streamedresult.Result {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sortChan, stopChan := css.launchSorters(list)

	streamedChan := make(chan streamedresult.Result, 1)
	go func() {
		resultChan := css.streamResults(sortChan, stopChan)
		var msg streamedresult.Result
		var ok bool
		for {
			select {
			case msg, ok = <-resultChan:
				if !ok {
					close(streamedChan)
					return
				}
				streamedChan <- msg
			case <-sigChan:
				// shutdown signal handling
				signal.Stop(sigChan)

				// stop sorters and wait
				close(stopChan)
				<-resultChan

				// return cancelled message
				msg.Err = ErrCancelled
				streamedChan <- msg

				// shutdown streaming channel
				close(streamedChan)
				return
			}
		}
	}()

	return streamedChan
}
