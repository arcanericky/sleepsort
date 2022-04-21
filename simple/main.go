package main

import (
	"fmt"

	"github.com/arcanericky/sleepsort/internal/simplesleepsort"
)

func main() {
	s := simplesleepsort.SimpleSleepSort{}
	items := []uint{1000, 900, 800, 700, 600, 500, 400, 300, 200, 100}
	fmt.Println("Input:", items)
	result, _ := s.Sort(items)
	fmt.Println("Result:", result)
}
