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
Goal is to simulate arbitrary mashmaps, and write out
the results for later analysis. Rewritten from my
original Python implementation.
*/

/*
TODO
-Update CSPRNG to use AES-NI & fallback to /dev/urandom or ChaCha20 if that fails
-Refactor to remove redundant variables & function calls
-Branch and swap map for radix tree
*/

// MapInstance stores a map + flags for solver settings + statistics
type MapInstance struct {
	diff, cols int
	ceiling    *big.Int
	record     []int
	debug      bool
	hashmap    []*cmap.ConcurrentMap
}

// UpdateInfo is passed to goroutines running GoUpdate.
type UpdateInfo struct {
	hashmap []*cmap.ConcurrentMap
	debug   bool
	cols    int
	ceiling *big.Int
}

// Diagnostic calculates the total # of hashes & prints debug info.
func (hmap MapInstance) Diagnostic() int {
	sum := 0
	for ind, val := range hmap.record {
		sum += val * (ind + 1)
	}
	if hmap.debug == true {
		fmt.Println("Diff is", hmap.diff)
		fmt.Println("Cols is", hmap.cols)
		fmt.Println("Record is", hmap.record)
		fmt.Println("Sum is", sum)
	}
	return sum
}

// RandString returns a psuedorandom string with value bounded by ceiling.
func RandString(ceiling *big.Int) string {
	rint, err := rand.Int(rand.Reader, ceiling)
	if err != nil {
		fmt.Println("CSPRNG Error!", err)
		// handle this error
	}
	return rint.String()
}

// GoUpdate is run by a goroutine to update the shared hashmap.
func GoUpdate(info UpdateInfo, sigkill chan bool) {
	defer waiter.Done()
	var absent bool
	var entry string
	flg := false
	// end condition is flg set to true by self or  by signal on sigkill
	for flg == false {
		select {
		case <-sigkill:
			flg = true
			return
		default:
			// generate random string & check if in first hashmap;
			// add entry if absent, else look up index
			entry = RandString(info.ceiling)
			absent = info.hashmap[0].SetIfAbsent(entry, 0)
			if absent == false {
				ind, _ := info.hashmap[0].Get(entry)
				indint := ind.(int)
				// Move entry forward a map & then check
				// if it is a solution
				info.hashmap[indint+1].Set(entry, true)
				// Catch out-of-bounds entries
				if indint+1 < info.cols-1 {
					info.hashmap[0].Set(entry, indint+1)
				}
				if indint != 0 {
					info.hashmap[indint].Remove(entry)
				}
				// Exit condition for solution found
				if indint+1 == info.cols-1 {
					if info.debug == true {
						fmt.Println("Desired collisions found!")
					}
					// Check sigkill before broadcasting completion
					// If first goroutine to finish broadcast & then set flg
					select {
					case <-sigkill:
						flg = true
						return
					default:
						for i := 0; i < runtime.NumCPU(); i++ {
							sigkill <- true
						}
						flg = true
						return
					}
				}
			}
		}
	}
}

// This version also uses the old multiple-hashmap method vs. a single map.

// Update is the single-threaded hashmap updating function left here for reference.
func (hmap MapInstance) Update() {
	// Set integer in first map, and use that as pointer
	// to most recent position for the entry
	flg := false
	absent := false
	var entry string
	for flg == false {
		entry = RandString(hmap.ceiling)
		absent = hmap.hashmap[0].SetIfAbsent(entry, 0)
		if absent == false {
			// We already know an entry exists so can ignore second return value
			ind, _ := hmap.hashmap[0].Get(entry)
			// Must cast out of interface{}
			indint := ind.(int)
			hmap.hashmap[indint+1].Set(entry, true)
			hmap.hashmap[0].Set(entry, indint+1)
			if indint != 0 {
				hmap.hashmap[indint].Remove(entry)
			}
			// Adjusting for 0-indexing (hmap.hashmap[1] is for double collisions)
			if indint+1 == hmap.cols-1 {
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
	hmap := MapInstance{
		diff, cols, ceiling, make([]int, cols),
		debug, make([]*cmap.ConcurrentMap, cols)}
	for i := range hmap.hashmap {
		temp := cmap.New()
		hmap.hashmap[i] = &temp
	}
	return hmap
}

// Sum aggregates statistics to pass to Diagnostic.
func (hmap MapInstance) Sum() int {
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

// Need to write results to file vs. pipe
// All command line options declared & parsed here

var diff int
var cols int
var iters int
var debug bool

func init() {
	// Defaults are diff 32 cols 3 iters 100 debug false
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
	runtime.GOMAXPROCS(runtime.NumCPU())
	outmap := make(map[int][]int)
	for i := 1; i < diff+1; i++ {
		for j := 2; j < cols+1; j++ {
			key := i*100 + j
			out := 0.0
			outcv := 0.0
			outmap[key] = make([]int, iters)
			for k := 0; k < iters; k++ {
				hmap := Hashmake(i, j, debug)
				waiter.Add(runtime.NumCPU())
				sigkill := make(chan bool, runtime.NumCPU()-1)
				for i := 0; i < runtime.NumCPU(); i++ {
					mappoint := make([]*cmap.ConcurrentMap, len(hmap.hashmap))
					copy(mappoint, hmap.hashmap)
					goinf := UpdateInfo{
						mappoint, hmap.debug,
						hmap.cols, hmap.ceiling}
					go GoUpdate(goinf, sigkill)
				}
				waiter.Wait()
				hashcount := hmap.Sum()
				outmap[key][k] = hashcount
				out += float64(hashcount)
			}
			out = out / float64(iters)
			for _, val := range outmap[key] {
				shift := float64(val) - out
				outcv += shift * shift
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
