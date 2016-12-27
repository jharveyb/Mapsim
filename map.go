package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"math"
	"flag"
	//"github.com/losalamos/rdrand"
	"github.com/orcaman/concurrent-map"
	"runtime"
	//"strconv"
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

var Updater cmap.UpsertCb
var Recordater cmap.UpsertCb

// Calculates the total # of hashes & prints useful info.

func (h Hashmap)  Diagnostic() int {
	sum := 0
	/*
	for i := uint(0); i < h.cols; i++ {
		inc, ok := h.record.Get(strconv.FormatInt(int64(i), 10))
		//fmt.Println(inc)
		if ok != false {
			incmap := inc.(MapEntry)
			sum += incmap.Count
		}
	}
	*/
	sum = int(h.cols)
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

// Updates hashmap with multiple threads until solved.
// Now only called once, so need to adjust.

func (h Hashmap) Update() {
	// going to do the single-threaded version first
	// desired behavior is to set in first map,
	// and use that entry as pointer to its most recent
	// position
	fmt.Println("Begin Update")
	flg := false
	absent := false
	entry := "test"
	for flg == false {
		entry = RandString(h.ceiling)
		//fmt.Println(entry)
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
				dubsum := 0
				for i := range h.hashmap {
					hitcount := h.hashmap[i].Count()
					//fmt.Println(hitcount)
					h.record[i] = hitcount
					//fmt.Println(h.hashmap[i].Items())
					if i != 0 {
						dubsum += hitcount
					}
				}
				h.record[0] -= dubsum
				flg = true
			}
		}
	}
	/*
	Updater = func(state bool, val interface{}, inpval interface{}) (out interface{}) {
		var outmap MapEntry
		var valmap MapEntry
		if state == true {
			valmap = val.(MapEntry)
			outmap.Count = valmap.Count + 1
			if uint(outmap.Count) == h.cols {
				if h.debug == true {
					fmt.Println("Desired collisions found!")
				}
				outmap.State = true
			}
		} else {
			outmap.Count = 1
		}
		out = outmap
		return out
	}
	*/
	/*
	Recordater = func(state bool, val interface{}, inpval interface{}) (out interface{}) {
		var outmap MapEntry
		outmap.Count = 1
		if state == true {
			outmap.Count += inpval.(int)
		}
		out = outmap
		return out
	}
	*/
	/*
	sol := h.hashmap.Upsert(entry, h.cols, Updater)
	solmap := sol.(MapEntry)
	if solmap.State == true {
		return true
	} else { return false }
	*/
	/*
	else {
		ind := solmap.Count - 1
		h.record.Upsert(strconv.FormatInt(int64(ind), 10), ind, Recordater)
		return false
	}
	*/
	/*
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
	*/
}

// Creates a hashmap & solves, returning the total # of hashes.

func Mapsim(diff int, cols int, debug bool) int {
	ceiling := new(big.Int).SetInt64(1 << uint(diff))
	hmap := Hashmap{
	diff, cols, ceiling, make([]int, cols),
	debug, make([]*cmap.ConcurrentMap, cols)}
	for i := range hmap.hashmap {
		temp := cmap.New()
		hmap.hashmap[i] = &temp
	}
	hmap.Update()
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
