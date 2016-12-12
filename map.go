package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"math"
	"flag"
	"runtime"
)

/* 
Goal is to simulate arbitrary mashmaps, and write out
the results for later analysis. Rewritten from my 
original Python implementation.
*/

type Hashmap struct {
	diff, cols int
	ceiling *big.Int
	record []int
	debug bool
	hashmap map[string]int
}

// Calculates the total # of hashes & prints useful info.

func (h Hashmap) Diagnostic() int {
	sum := 0
	for i := 0; i < h.cols; i++ {
		sum += h.record[i]
	}
	if h.debug == true {
		fmt.Println("Diff is", h.diff)
		fmt.Println("Cols is", h.cols)
		fmt.Println("Record is", h.record)
		fmt.Println("Sum is", sum)
	}
	return sum
}

// Updates hashmap & returns state of problem solving.

func (h Hashmap) Update() bool {
	rint, err := rand.Int(rand.Reader, h.ceiling)
	if err != nil {
		fmt.Println("CSPRNG Error!", err)
		// handle this error
	}
	entry := rint.String()
	result := h.Check(entry, 1)
	return result
}

func (h Hashmap) Check(entry string, inc int) bool {
	fmt.Println("h", h, "entry", entry, "inc", inc)
	ind, ok := h.hashmap[entry]
	if ok == true {
		h.record[ind] += inc
		h.hashmap[entry] += inc
		if h.hashmap[entry] == h.cols {
			if h.debug == true {
				fmt.Println("Desired collisions found!")
				fmt.Println(h)
			}
			return true
		}
	}
	if ok == false {
		h.hashmap[entry] = inc
		h.record[0] += inc
	}
	return false
}

// Given a hashmap, fills up to count and returns. Communicates about solutions and progress.

func Worker(h Hashmap, count int) Hashmap {
	fmt.Println(h)
	hflg := false
	for i := 0; i < count; i++ {
		hflg = h.Update()
		i += 1
		if hflg == true {
			break
		}
	}
	return h
}

// Creates a hashmap & solves, returning the total # of hashes.

func Mapsim(diff int, cols int, debug bool) int {
	ceiling := new(big.Int).SetUint64(1 << uint(diff))
	/*
	hmap := Hashmap{
	diff, cols, ceiling, make([]int, cols),
	debug, make(map[string]int)}
	hflg := false
	for hflg == false {
		hflg = hmap.Update()
	}
	return hmap.Diagnostic()
	*/
	hflg := false
	count := int(1e4)
	core := runtime.NumCPU()
	hmap := Hashmap{
	diff, cols, ceiling, make([]int, cols),
	debug, make(map[string]int)}
	cache := make([]Hashmap, core)
	for i := 0; i < core; i++ {
	}
	for hflg == false {
		for i := 0; i < core; i++ {
			cache[i] = Worker(Hashmap{
			diff, cols,
			new(big.Int).SetUint64(1 << uint(diff)),
			make([]int, cols), debug,
			make(map[string]int)},
			count)
			fmt.Println(i)
		}
		fmt.Println("hmap", hmap)
		for i := 0; i < core; i++ {
			newmap := cache[i]
			fmt.Println("newmap", newmap)
			for key, val := range newmap.hashmap {
				hflg = hmap.Check(key, val)
				if hflg == true {
					fmt.Println("Breaking merge!")
					break
				}
			}
			if hflg == true {
				break
			}
			fmt.Println("hmap", hmap)
		}
		fmt.Println(hmap)

	}
	return hmap.Diagnostic()
}

// Need to write results to file vs. pipe, make concurrent
// All command line options declared & parsed here
var diff int
var cols int
var iters int
var debug bool

func init() {
	flag.IntVar(&diff, "diff", 32, "Difficulty of the PoW")
	flag.IntVar(&cols, "cols", 3, "# of Collisions")
	flag.IntVar(&iters, "iters", 100, "# of iterations per parameter")
	flag.BoolVar(&debug, "debug", false, "Sets state of printing while solving")
	}
/*
Parses flags, & solves PHC PoW over the range specified with
diffs & cols, running for iters # of times. Prints diffs & cols,
along with the mean, s.d., & coeff. of var.
*/

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	outmap := make(map[int][]int)
	for i := 1; i < diff+1; i++ {
		for j := 2; j < cols+1; j++ {
			key := i*100 + j
			out := 0.0
			outcv := 0.0
			outmap[key] = make([]int, iters)
			for k := 0; k < iters; k++ {
				hashcount := Mapsim(i, j, debug)
				outmap[key][k] = hashcount
				out += float64(hashcount)
			}
			out = out / float64(iters)
			for _, val := range outmap[key] {
				shift := float64(val)-out
				outcv += shift*shift
			}
			outcv = outcv / float64(iters)
			outcv = math.Sqrt(outcv)
			outcv = outcv / out
			fmt.Println(key)
			fmt.Println(out)
			fmt.Println(outcv)
		}
	}
}
