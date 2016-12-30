package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"math"
	"flag"
	"github.com/orcaman/concurrent-map"
	"runtime"
	"sync"
	/*
	"strconv"
	"github.com/losalamos/rdrand"
	*/
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
	hashmap []*cmap.ConcurrentMap
}

type UpdateInfo struct {
	hashmap []*cmap.ConcurrentMap
	debug bool
	cols int
	ceiling *big.Int
}

// Calculates the total # of hashes & prints useful info.

func (h Hashmap) Diagnostic() int {
	sum := 0
	for ind, val := range h.record {
		sum += val*(ind+1)
	}
	if h.debug == true {
		fmt.Println("Diff is", h.diff)
		fmt.Println("Cols is", h.cols)
		fmt.Println("Record is", h.record)
		fmt.Println("Sum is", sum)
	}
	return sum
}

func RandString(ceiling *big.Int) string {
	rint, err := rand.Int(rand.Reader, ceiling)
	if err != nil {
		fmt.Println("CSPRNG Error!", err)
		// handle this error
	}
	return rint.String()
}

// Function for a goroutine to update the global hashmap;
// A subset of the original update function, using 
// UpdateInfo to keep data passing clean

func GoUpdate(info UpdateInfo, waiter sync.WaitGroup) {
	defer waiter.Done()
	var absent bool
	var entry string
	flg := false
	for flg == false {
		entry = RandString(info.ceiling)
		absent = info.hashmap[0].SetIfAbsent(entry, 0)
		if absent == false {
			ind, _ := info.hashmap[0].Get(entry)
			indint := ind.(int)
			if indint+1 == info.cols-1 {
				if info.debug ==  true {
					fmt.Println("Desired collisions found!")
				}
				flg = true
			} else {
				info.hashmap[indint+1].Set(entry, true)
				info.hashmap[0].Set(entry, indint+1)
				if indint != 0 {
					info.hashmap[indint].Remove(entry)
				}
			}
		}
	}
}

// Updates hashmap with multiple threads until solved.
// Now only called once, so need to adjust.

func (h Hashmap) Update() {
	// going to do the single-threaded version first
	// desired behavior is to set in first map,
	// and use that entry as pointer to its most recent
	// position
	flg := false
	absent := false
	var entry string
	for flg == false {
		entry = RandString(h.ceiling)
		absent = h.hashmap[0].SetIfAbsent(entry, 0)
		if absent == false {
			ind, _ := h.hashmap[0].Get(entry)
			indint := ind.(int)
			h.hashmap[indint+1].Set(entry, true)
			h.hashmap[0].Set(entry, indint+1)
			if indint != 0 {
				h.hashmap[indint].Remove(entry)
			}
			if indint+1 == h.cols-1 {
				if h.debug ==  true {
					fmt.Println("Desired collisions found!")
				}
				flg = true
			}
		}
	}
}

// Creates a hashmap & solves, returning the total # of hashes.

func Hashmake(diff int, cols int, debug bool) Hashmap {
	ceiling := new(big.Int).SetInt64(1 << uint(diff))
	hmap := Hashmap{
	diff, cols, ceiling, make([]int, cols),
	debug, make([]*cmap.ConcurrentMap, cols)}
	for i := range hmap.hashmap {
		temp := cmap.New()
		hmap.hashmap[i] = &temp
	}
	return hmap
}

func (hmap Hashmap) Sum() int {
	dubsum := 0
	for i := range hmap.hashmap {
		curmap := hmap.hashmap[i]
		hitcount := curmap.Count()
		hmap.record[i] = hitcount
		if i != 0 {
			dubsum += hitcount
		}
	}
	hmap.record[0] -= dubsum
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
	var waiter sync.WaitGroup
	for i := 1; i < diff+1; i++ {
		for j := 2; j < cols+1; j++ {
			key := i*100 + j
			out := 0.0
			outcv := 0.0
			outmap[key] = make([]int, iters)
			for k := 0; k < iters; k++ {
				// move mapsim logic into main
				hmap := Hashmake(i, j, debug)
				waiter.Add(runtime.NumCPU())
				for i := 0; i < runtime.NumCPU(); i++ {
					mappoint := make([]*cmap.ConcurrentMap, len(hmap.hashmap))
					copy(mappoint, hmap.hashmap)
					goinf := UpdateInfo{
					mappoint, hmap.debug,
					hmap.cols, hmap.ceiling}
					go GoUpdate(goinf, waiter)
				}
				fmt.Println("All routines made!")
				waiter.Wait()
				fmt.Println("All routines made!")
				//end
				hashcount := hmap.Sum()
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
