# Sleep Sort

My implementation of the [Sleep Sort][sleep-sort] in Go.

## Background

A friend introduced me to the concept of the sleep sort. After reading
about it, I had to try implementing it in Go using Goroutines and
channels. It ended up being one of my more fun toy projects and now I
want to share.

## Quick Start

Some sorts can take many minutes to complete. To exit early, use Ctrl+C.

```go
To quickly execute using default values, use

```shell
go run . simple
```

```shell
go run . sequential
```

and

```shell
go run . concurrent
```

For help, execute with:

```shell
go run . --help
```

And for help specific to each command (`sequential`, `concurrent`, and
`simple`), execute with (example):

```shell
go run . concurrent --help
```

### Output

The output will contain the type of sort being executed along with the number of items and additional information if applicable to the sort type. A list of the unsorted items will be displayed and then the sort is executed.

```text
Sleep sorting 100 items concurrently using 7 rounds
Unsorted items: [8081 7887 1847 4059 2081 1318 4425 2540 456 3300 694 8511 8162 5089 4728 3274 1211 1445 3237 9106 495 5466 1528 6258 8047 9947 8287 2888 2790 3015 5541 408 7387 6831 5429 5356 1737 631 1485 5026 6413 3090 5194 563 2433 4147 4078 4324 6159 1353 1957 3721 7189 2199 3000 8705 2888 4538 9703 9355 2451 8510 2605 156 8266 9828 5561 7202 4783 5746 1563 4376 9002 9718 5447 5094 1577 7463 7996 6420 8623 953 1137 3133 9241 59 3033 8643 3891 2002 8878 9336 2546 9107 7940 6503 552 9843 2205 1598]
```

When the sort is executed, multiple Goroutines will be launched. The display converts to a spinner to indicate the Goroutines are waited upon along with the number of Goroutines that are running. As the Goroutines complete, the displayed number of routines will decrement.

```text
⣯ Waiting for 690 Goroutines
```

As the sorting "rounds" complete, the elapsed time is displayed along with a list of mismatches from the two most recent sorts.

```text
Round 2 sorted in 10 seconds
Unsorted items (2):
8511 != 8510    8510 != 8511
```

Once the sort is complete, a list of sorted items is displayed along with an execution summary.

```text
Sorted items: [59 156 408 456 495 552 563 631 694 953 1137 1211 1318 1353 1445 1485 1528 1563 1577 1598 1737 1847 1957 2002 2081 2199 2205 2433 2451 2540 2546 2605 2790 2888 2888 3000 3015 3033 3090 3133 3237 3274 3300 3721 3891 4059 4078 4147 4324 4376 4425 4538 4728 4783 5026 5089 5094 5194 5356 5429 5447 5466 5541 5561 5746 6159 6258 6413 6420 6503 6831 7189 7202 7387 7463 7887 7940 7996 8047 8081 8162 8266 8287 8510 8511 8623 8643 8705 8878 9002 9106 9107 9241 9336 9355 9703 9718 9828 9843 9947]
Sort complete in 40 seconds
Goroutines: 2
```

## Basic Sort

The basic sort is performed with the `SleepSort` class. It takes in an
array of uints and sorts it using sleeping Goroutines, each sleeping
for a time equal to the uint value multiplied by one millisecond
multiplied by a multiplier. The multiplier is a feature that is useful
when the input values are closely spaced. Because of sleep sorting
limitations, the output may not be fully sorted.

## Simple Sort

The most simple implementation is with the `SimpleSleepSort` class.
Reading the code for
[`SimpleSleepSort`](internal/simplesleepsort/simplesleepsort.go) is a
good introduction to the basis for the more complex implementations.
It sleep sorts using a single iteration (non-validating) and returns
the sorted array. In the spirit of keeping the code minimal, the
`main` package is implemented in `simple/main.go` with a hardcoded
array of values and calls the `Sort` method to perform the sort.
Execute it with:

```shell
go run ./simple/...
```

For completeness, the simple implementation is also included in the
main program and can be ran with the `simple` command.

```shell
go run . simple
```

## Validating Sorts

To address sleep sorting limitations, some validation functionality
has been built on top of the basic sort. This validation executes the
sort multiple times, each time using an increased multiplier value.
After each sort, a comparison is made between the two most recent
sorts and if they match, the result is returned. Because it's still
possible that two sleep sorts could return invalid sorts, even a
validating sort may return incorrect output. It's the nature of the
sleep sort.

### Sequential Validating Sort

The sequential validating sort launches the sleep sort routines
sequentially, each time doubling the multiplier until the sorted
output from two runs match. The exception is the first two sorts which
are performed with a multiplier of 1. These sorts are executed
concurrently then compared, after which each additional sort is
performed sequentially. This will always yield a validated (but not
guaranteed) sort because it loops until the two most recent runs
match.

Execute this sort with a command such as:

```shell
go run sleepsort.go sequential --items 100
```

### Concurrent Validating Sort

After coding the first two sorts to execute concurrently, I
also realized that all sorts can be performed concurrently to save
time. I quit working on the Sequential Validating Sort and began
building the Concurrent Validating Sort.

This sort takes a number of rounds as input and launches a
corresponding sort for each round, with each round having its
multiplier doubled. The exception is the first round, of which two
sorts are ran as Goroutines to quickly get an initial result to
compare.

Unlike the sequential sort which will always yield a validated result,
the maximum round count means two sorts may not pass the validation
test by the time the maximum round count is reached and the sort will
fail, returning an error.

Execute with:

```shell
go run sleepsort.go concurrent --items 100 --rounds 8
```

An interesting thing to keep mind about this implementation is the
amount of Goroutines that are launched. By default, it launches 8
rounds (the first round is ran twice, then once for each additional
round), each with 100 items in the list. That's 800 Goroutines. Add in
the managing Goroutine for each round plus a few more for handling
other things and it's running over 810 Goroutines with the number
reducing as the result of each sleep rolls in. It's fun to launch it
with settings such as 1,000 items and 20 Goroutines (`go run .
concurrent --items 1000 --rounds 20`) and watch the number of
mismatches slowly dwindle as the code arrives at an answer.

## List Value Generation

By default the list of values is generated by an unseeded
pseudo-random number generator from `math/rand` rather than
`crypto/rand`. This helps with testing and experimentation because
these values are repeatable. To seed the generator with the current
time, use the `--seed` flag.

## Runtime Metrics

This program implements the [`expvar` package][expvar] to provide runtime metrics. Access it with `http://localhost:8080/debug/vars`.

```shell
$ curl -s 127.0.0.1:8080/debug/vars | jq .
{
  "cmdline": [
    "/tmp/go-build127810536/b001/exe/sleepsort",
    "concurrent",
    "--items",
    "300"
  ],
  "memstats": {
    "Alloc": 2032560
...
```

## Contributing

You'll surely spot defects (especially in the use of Goroutines and channels) and notice algorithm improvements or maybe even have a different technique other than sequential or concurrent. I'd like to see your PR.

[expvar]: https://pkg.go.dev/expvar
[sleep-sort]: https://www.geeksforgeeks.org/sleep-sort-king-laziness-sorting-sleeping/
