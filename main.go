package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/hako/durafmt"
	"github.com/spf13/cobra"

	_ "expvar"

	"github.com/arcanericky/sleepsort/internal/concurrent"
	"github.com/arcanericky/sleepsort/internal/sequential"
	"github.com/arcanericky/sleepsort/internal/simplesleepsort"
	"github.com/arcanericky/sleepsort/internal/streamedresult"
)

var (
	version = "dev"
)

type StreamedSleepSorter interface {
	StreamSort([]uint) chan streamedresult.Result
}

type SimpleSortAdapter struct {
	simplesleepsort.SimpleSleepSort
}

func (ss SimpleSortAdapter) StreamSort(list []uint) chan streamedresult.Result {
	resultChan := make(chan streamedresult.Result, 1)

	go func() {
		startTime := time.Now()
		result, _ := ss.Sort(list)
		resultChan <- streamedresult.New(1, startTime, time.Now(), nil, result, nil)
		close(resultChan)
	}()

	return resultChan
}

func gatherUnsorted(left []uint, right []uint) []int {
	mismatchList := []int{}
	for i := range left {
		if left[i] != right[i] {
			mismatchList = append(mismatchList, i)
		}
	}

	return mismatchList
}

func showUnsorted(left []uint, right []uint) {
	mismatchList := gatherUnsorted(left, right)
	if len(mismatchList) == 0 {
		return
	}
	fmt.Printf("Unsorted items (%d):\n", len(mismatchList))

	const tab = "\t"
	const newline = "\n"
	tw := new(tabwriter.Writer)
	tw.Init(os.Stdout, 15, 15, 0, '\t', 0)
	defer tw.Flush()

	j := 0
	var separator string
	for _, i := range mismatchList {
		j++
		switch j {
		case 4:
			separator = newline
			j = 0
		default:
			separator = tab
		}
		fmt.Fprintf(tw, "%d != %d%s", left[i], right[i], separator)
	}

	if separator != newline {
		fmt.Fprint(tw, newline)
	}
}

// generateItems will (pseudo) randomly generate a list of uints
func generateItems(itemCount uint, seed bool) []uint {
	list := make([]uint, itemCount)

	if seed {
		rand.Seed(time.Now().UnixNano())
	}

	for i := range list {
		list[i] = uint(rand.Intn(10000))
	}

	return list
}

func newSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[11], 250*time.Millisecond)
	s.Reverse()
	s.Color("yellow")
	return s
}

func spinnerUpdater(s *spinner.Spinner) (chan struct{}, *sync.WaitGroup) {
	var wg sync.WaitGroup
	sc := make(chan struct{}, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-sc:
				return
			default:
				s.Lock()
				s.Suffix = fmt.Sprintf(" Waiting for %d Goroutines", runtime.NumGoroutine())
				s.Unlock()
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return sc, &wg
}

func streamSort(itemCount uint, seed bool, sorter StreamedSleepSorter) error {
	netSock, netErr := net.Listen("tcp", "127.0.0.1:8080")
	if netErr != nil {
		fmt.Fprintln(os.Stdout, netErr)
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprint(os.Stderr, "Failed to start metrics server")
			}
		}()
		http.Serve(netSock, nil)
	}()

	items := generateItems(itemCount, seed)
	fmt.Println("Unsorted items:", items)
	now := time.Now()
	resultChan := sorter.StreamSort(items)
	var lastSortedList []uint
	var lastErr error

	waitSpinner := newSpinner()
	updaterStopChan, updaterWg := spinnerUpdater(waitSpinner)
	waitSpinner.Start()
	for msg := range resultChan {
		waitSpinner.Stop()
		lastErr = msg.Err
		if msg.Err != nil {
			fmt.Println("Sort stopped with:", msg.Err)
			break
		}

		fmt.Printf("Round %d sorted in %s\n", msg.Round,
			durafmt.Parse(msg.EndTime.Sub(msg.BeginTime).Round(time.Second)))
		waitSpinner.Start()

		showUnsorted(msg.PreviousResult, msg.CurrentResult)
		lastSortedList = msg.CurrentResult
	}

	waitSpinner.Stop()
	elapsed := durafmt.Parse(time.Since(now).Round(time.Second))
	close(updaterStopChan)
	updaterWg.Wait()
	fmt.Println("Sorted items:", lastSortedList)
	fmt.Println("Sort complete in", elapsed)
	fmt.Println("Goroutines:", runtime.NumGoroutine())

	if netErr == nil {
		netSock.Close()
	}
	return lastErr
}

// run is the main program logic
func run() int {
	retVal := 0

	var rootCmd = &cobra.Command{
		Use:     "sleepsort",
		Short:   "Sleep Sort lazily sorts using sleepy time",
		Version: version,
	}
	var itemCount uint
	var seed bool
	rootCmd.PersistentFlags().UintVar(&itemCount, "items", 100, "Number of items to generate")
	rootCmd.PersistentFlags().BoolVar(&seed, "seed", false, "Seed random generator with current time")

	var rounds uint
	var concurrentCmd = &cobra.Command{
		Use:   "concurrent",
		Short: "Execute sleep sorts concurrently",
		Run: func(cmd *cobra.Command, args []string) {
			suffix := "s"
			if rounds == 1 {
				suffix = ""
			}
			fmt.Printf("Sleep sorting %d items concurrently using %d round%s\n", itemCount, rounds, suffix)
			if streamSort(itemCount, seed, concurrent.ConcurrentSleepSort{MaxRounds: rounds}) != nil {
				retVal = 1
			}
		},
	}
	concurrentCmd.Flags().UintVar(&rounds, "rounds", 7, "Number of sleep sort rounds to run")
	rootCmd.AddCommand(concurrentCmd)

	var sequentialCmd = &cobra.Command{
		Use:   "sequential",
		Short: "Execute sleep sorts sequentially",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Sleep sorting %d items sequentially\n", itemCount)
			if streamSort(itemCount, seed, sequential.SequentialSleepSort{}) != nil {
				retVal = 1
			}
		},
	}
	rootCmd.AddCommand(sequentialCmd)

	var simpleCmd = &cobra.Command{
		Use:   "simple",
		Short: "Execute a single sleep sort using a simple implementation",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Sleep sorting %d items simply with a single round\n", itemCount)
			streamSort(itemCount, seed, SimpleSortAdapter{})
		},
	}
	rootCmd.AddCommand(simpleCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		retVal = 1
	}

	return retVal
}

func main() {
	os.Exit(run())
}
