package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"math"
	"flag"
)

/* 
Goal is to simulate arbitrary mashmaps, and write out
the results for later analysis. Rewritten from my 
original Python implementation.
*/

type Hashmap struct {
	diff, cols uint
	ceiling *big.Int
	record []uint
	debug bool
	hashmap map[string]uint
}

// Calculates the total # of hashes & prints useful info.

func (h Hashmap) Diagnostic() uint {
	sum := uint(0)
	for i := uint(0); i < h.cols; i++ {
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
	ind, ok := h.hashmap[entry]
	if ok == true {
		h.record[ind] += 1
		h.hashmap[entry] += 1
		if h.hashmap[entry] == h.cols {
			if h.debug == true {
				fmt.Println("Desired collisions found!")
			}
			return true
		}
	}
	if ok == false {
		h.hashmap[entry] = 1
		h.record[0] += 1
	}
	return false
}

// Creates a hashmap & solves, returning the total # of hashes.

func Mapsim(diff uint, cols uint, debug bool) uint {
	ceiling := new(big.Int).SetUint64(1 << diff)
	hmap := Hashmap{
	diff, cols, ceiling, make([]uint, cols),
	debug, make(map[string]uint)}
	hflg := false
	for hflg == false {
		hflg = hmap.Update()
	}
	return hmap.Diagnostic()
}

// Need to write results to file vs. pipe, make concurrent
// All command line options declared & parsed here

var diff uint
var cols uint
var iters uint
var debug bool

func init() {
	flag.UintVar(&diff, "diff", 32, "Difficulty of the PoW")
	flag.UintVar(&cols, "cols", 3, "# of Collisions")
	flag.UintVar(&iters, "iters", 100, "# of iterations per parameter")
	flag.BoolVar(&debug, "debug", false, "Sets state of printing while solving")
	}

/*
Parses flags, & solves PHC PoW over the range specified with
diffs & cols, running for iters # of times. Prints diffs & cols,
along with the mean, s.d., & coeff. of var.
*/

func main() {
	flag.Parse()
	outmap := make(map[uint][]uint)
	for i := uint(1); i < diff+1; i++ {
		for j := uint(2); j < cols+1; j++ {
			key := uint(i*100 + j)
			out := 0.0
			outcv := 0.0
			outmap[key] = make([]uint, iters)
			for k := uint(0); k < iters; k++ {
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
