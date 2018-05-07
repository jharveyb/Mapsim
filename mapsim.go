package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sync"

	"github.com/orcaman/concurrent-map"
	/*
		"strconv"
		"github.com/losalamos/rdrand"
	*/)

/*
Goal is to simulate arbitrary mashmaps, and write out the results for later analysis.
Rewritten from my original Python implementation.
*/

/*
TODO
-Update RandString to use AES-NI & fallback to /dev/urandom or ChaCha20 if that fails
-Branch and swap map for radix tree
*/

// Note that the new sync.Map is not appropriate here since we're changing values frequently.

// MapInstance stores an array of maps, solver settings, and an array for statistics.
type MapInstance struct {
	diff, cols int
	ceiling    *big.Int
	record     []int
	debug      bool
	hashmap    []*cmap.ConcurrentMap
}

// Sum populates the record array with statistics and calls Diagnostic.
func (hmap MapInstance) Sum() int {
	var entrycount int
	for i := range hmap.hashmap {
		entrycount = hmap.hashmap[i].Count()
		hmap.record[i] = entrycount
		// Must adjust for the pointers in the first map
		if i > 0 {
			hmap.record[0] -= entrycount
		}
	}
	return hmap.Diagnostic()
}

// Diagnostic calculates the total # of hashes & can print more detailed statistics.
func (hmap MapInstance) Diagnostic() int {
	sum := 0
	for index, val := range hmap.record {
		sum += val * (index + 1)
	}
	if hmap.debug == true {
		fmt.Println("Diff is", hmap.diff)
		fmt.Println("Cols is", hmap.cols)
		fmt.Println("Record is", hmap.record)
		fmt.Println("Sum is", sum)
	}
	return sum
}

// RandString returns a psuedorandom string with the value bounded by ceiling.
func RandString(ceiling *big.Int) string {
	rint, err := rand.Int(rand.Reader, ceiling)
	if err != nil {
		fmt.Println("CSPRNG Error!", err)
		// handle this error
	}
	return rint.String()
}

// GoUpdate is run by a goroutine to update the shared hashmap.
func GoUpdate(hmap MapInstance, sigkill, sigdone chan bool) {
	defer waiter.Done()
	var absent bool
	var entry string
	var index interface{}
	var indexint int
	flg := false
	// end condition is finding solution or signal on sigkill
	for flg == false {
		select {
		case <-sigkill:
			flg = true
		default:
			// generate random string & check if in first hashmap;
			// add entry if absent, else look up index
			entry = RandString(hmap.ceiling)
			absent = hmap.hashmap[0].SetIfAbsent(entry, 0)
			if absent == false {
				index, _ = hmap.hashmap[0].Get(entry)
				indexint = index.(int)
				// Move entry forward a map
				hmap.hashmap[indexint+1].Set(entry, true)
				// Catch out-of-bounds pointers
				if indexint+1 < hmap.cols-1 {
					hmap.hashmap[0].Set(entry, indexint+1)
				}
				// Only remove entry if this is not the first collision
				if indexint != 0 {
					hmap.hashmap[indexint].Remove(entry)
				}
				// Solution found
				if indexint+1 == hmap.cols-1 {
					if hmap.debug == true {
						fmt.Println("Desired collisions found!")
					}
					// Send message that solution is found before terminating
					sigdone <- true
					flg = true
				}
			}
		}
	}
}

// Update is the single-threaded hashmap updating function left here for reference.
func (hmap MapInstance) Update() {
	// Set integer in first map, and use that as pointer
	// to most recent position for the entry
	var entry string
	var absent bool
	flg := false
	for flg == false {
		entry = RandString(hmap.ceiling)
		absent = hmap.hashmap[0].SetIfAbsent(entry, 0)
		if absent == false {
			// We already know an entry exists so can ignore second return value
			index, _ := hmap.hashmap[0].Get(entry)
			indexint := index.(int)
			hmap.hashmap[indexint+1].Set(entry, true)
			hmap.hashmap[0].Set(entry, indexint+1)
			if indexint != 0 {
				hmap.hashmap[indexint].Remove(entry)
			}
			// Adjusting for 0-indexing (hmap.hashmap[1] is for double collisions)
			if indexint+1 == hmap.cols-1 {
				if hmap.debug == true {
					fmt.Println("Desired collisions found!")
				}
				flg = true
			}
		}
	}
}

// Hashmake creates & initializes a MapInstance to be passed to goroutines.
func Hashmake(diff int, cols int, debug bool) MapInstance {
	ceiling := new(big.Int).SetInt64(1 << uint(diff))
	hmap := MapInstance{diff, cols, ceiling, make([]int, cols),
		debug, make([]*cmap.ConcurrentMap, cols)}
	for i := range hmap.hashmap {
		temp := cmap.New()
		hmap.hashmap[i] = &temp
	}
	return hmap
}

// Need to write results to file vs. pipe
// All command line options declared & parsed here

var diff int
var cols int
var iters int
var debug bool

func init() {
	flag.IntVar(&diff, "diff", 32, "Difficulty of the PoW (n)")
	flag.IntVar(&cols, "cols", 3, "# of Collisions (k)")
	flag.IntVar(&iters, "iters", 100, "# of iterations per (n, k)")
	flag.BoolVar(&debug, "debug", false, "Sets state of printing while solving")
}

/*
Parses flags, & solves PHC PoW over the range specified with
diffs & cols, running for iters # of times. Prints diffs & cols,
along with the mean, s.d., & coeff. of var.
*/

// Must be global
var waiter sync.WaitGroup

func main() {
	flag.Parse()
	cpucount := runtime.NumCPU()
	runtime.GOMAXPROCS(cpucount)
	outputmap := make(map[int][]int)
	var key int
	var mean, coeffvar, offset float64
	var hmap MapInstance
	var sigkill, sigdone chan bool
	var hashcount int
	if iters < 2 {
		fmt.Println("Cannot calculate coeff. of var. with < 2 iterations.")
	}
	for diffindex := 1; diffindex < diff+1; diffindex++ {
		for colindex := 2; colindex < cols+1; colindex++ {
			// Implicit limit of 64 for difficulty from size of Int64
			key = colindex*100 + diffindex
			mean, coeffvar = 0.0, 0.0
			outputmap[key] = make([]int, iters)
			for iterindex := 0; iterindex < iters; iterindex++ {
				hmap = Hashmake(diffindex, colindex, debug)
				sigkill, sigdone = make(chan bool, cpucount), make(chan bool, cpucount)
				waiter.Add(cpucount)
				for cpuindex := 0; cpuindex < cpucount; cpuindex++ {
					go GoUpdate(hmap, sigkill, sigdone)
				}
				go func() {
					select {
					case <-sigdone:
						for cpuindex := 0; cpuindex < cpucount; cpuindex++ {
							sigkill <- true
						}
					}
				}()
				waiter.Wait()
				hashcount = hmap.Sum()
				outputmap[key][iterindex] = hashcount
				mean += float64(hashcount)
			}
			// Calculate mean, std. dev., and then coeff. of var.
			mean /= float64(iters)
			for _, val := range outputmap[key] {
				offset = float64(val) - mean
				coeffvar += offset * offset
			}
			coeffvar /= float64(iters - 1)
			coeffvar = math.Sqrt(coeffvar)
			coeffvar /= mean
			if debug == true {
				fmt.Println("Collisions + Difficulty: ", key)
				fmt.Println("Mean # of hashes to solve: ", mean)
				if iters > 1 {
					fmt.Println("Coefficient of variation: ", coeffvar)
				}
			} else {
				fmt.Println(key)
				fmt.Println(mean)
				if iters > 1 {
					fmt.Println(coeffvar)
				}
			}
		}
	}
}
