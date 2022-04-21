package simplesleepsort

import (
	"sync"
	"time"
)

// SimpleSleepSort is a class that will sleep sort a list of uints using
// Goroutines to sleep in milliseconds corresponding to the value of
// the uint multiplied by the multiplier. The output is not guaranteed
// to be sorted.
type SimpleSleepSort struct{}

func (s SimpleSleepSort) scatter(list []uint) chan uint {
	sortChan := make(chan uint, 1)

	var wg sync.WaitGroup
	wg.Add(len(list))
	for _, entry := range list {
		go func(i uint) {
			defer wg.Done()
			time.Sleep(time.Millisecond * time.Duration(i))
			sortChan <- i
		}(entry)
	}

	go func() {
		wg.Wait()
		close(sortChan)
	}()

	return sortChan
}

func (s SimpleSleepSort) gather(sortChan chan uint, numItems int) []uint {
	var sortedList []uint = make([]uint, numItems)

	for i := 0; i < numItems; i++ {
		sortedList[i] = <-sortChan
	}

	return sortedList
}

func (s SimpleSleepSort) Sort(list []uint) ([]uint, error) {
	return s.gather(s.scatter(list), len(list)), nil
}
