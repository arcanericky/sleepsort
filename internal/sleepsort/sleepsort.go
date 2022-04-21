package sleepsort

import (
	"sync"
	"time"
)

// SleepSort is a class that will sleep sort a list of uints using
// Goroutines to sleep in milliseconds corresponding to the value of
// the uint multiplied by the multiplier. The output is not guaranteed
// to be sorted.
type SleepSort struct {
	Multiplier uint
}

func (s SleepSort) scatter(list []uint) (chan uint, chan struct{}) {
	sortChan := make(chan uint, 1)
	stopChan := make(chan struct{}, 1)

	var wg sync.WaitGroup
	wg.Add(len(list))
	for _, entry := range list {
		go func(i uint) {
			defer wg.Done()
			const interval = 50 * time.Millisecond
			duration := time.Millisecond * time.Duration(i*s.Multiplier)
			now := time.Now()

			for time.Since(now) < duration {
				select {
				case <-stopChan:
					return
				default:
				}

				remaining := duration - time.Since(now)
				switch {
				case remaining > time.Duration(interval):
					time.Sleep(interval)
				case remaining > 0:
					time.Sleep(remaining)
				}
			}

			select {
			case <-stopChan:
				return
			default:
			}

			sortChan <- i
		}(entry)
	}

	go func() {
		wg.Wait()
		close(sortChan)
	}()

	return sortChan, stopChan
}

func (s SleepSort) gather(sortChan chan uint, stopChan chan struct{}) []uint {
	var sortedList []uint

	for {
		select {
		case <-stopChan:
			return sortedList
		default:
		}

		entry, ok := <-sortChan
		if !ok {
			return sortedList
		}

		sortedList = append(sortedList, entry)
	}
}

func (s SleepSort) Sort(list []uint) (chan []uint, chan struct{}) {
	resultChan := make(chan []uint, 1)

	sortChan, stopChan := s.scatter(list)
	go func() {
		resultChan <- s.gather(sortChan, stopChan)
	}()

	return resultChan, stopChan
}
