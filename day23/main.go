package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"
)

// State representation. I should probably haved used a smaller structure for the hallway
// but is was easier for me to figure out what to do using this one.
// 		h: hallway
// 		mobile: number of misplaced amphipods in each room
// 		empty: number of empty spots in each room
// 		cost: current cost
// 		remaining: total number of misplaced amphipods (rooms & hallway)
type State struct {
	h         [11]uint8
	mobile    [4]uint8
	empty     [4]uint8
	cost      uint32
	remaining uint8
}

// Verify that we are allowed to move between two hallway spots.
func (s *State) pathClear(src uint8, dst uint8) bool {

	// Two loops seem faster than swapping values (src, dst = dst, src).
	if src > dst {
		for c := dst + 1; c < src; c++ {
			if s.h[c] != free {
				return false
			}
		}
	} else {
		for c := src + 1; c < dst; c++ {
			if s.h[c] != free {
				return false
			}
		}
	}

	return true

}

type Key struct {
	h      [11]uint8
	mobile [4]uint8
	empty  [4]uint8
}

const free = 4

var (
	experimental bool
	spots        [7]uint8 = [7]uint8{0, 1, 3, 5, 7, 9, 10}
	m            [][4]uint8
	stack        []State
	visited      map[Key]uint32
	target       uint32      = 0
	minCosts     [][5]uint32 = [][5]uint32{
		{0, 2, 5, 10, 17},
		{0, 20, 50, 110, 180},
		{0, 200, 500, 1100, 1800},
		{0, 2000, 5000, 10000, 17000},
	}

	// Probably faster than Math.Pow10() / switch / if-else statements
	weights [4]uint32 = [4]uint32{1, 10, 100, 1000}
)

// Read + parse input from file, call processInput once or twice according to the input type
// and print the answer(s).
func main() {

	start := time.Now()

	var file string

	flag.StringVar(&file, "f", "input.txt", "Input file")
	flag.BoolVar(&experimental, "e", false, "Enable experimental features")
	flag.Parse()

	input, _ := os.ReadFile(file)

	re := regexp.MustCompile(`[^ABCD]`)
	chars := re.ReplaceAll(input, []byte(""))

	if len(chars) == 8 {
		p1, t1 := processInput(chars)
		fmt.Printf("Part1: found cost of %d in %s\n", p1, t1)
	} else {
		var part1 []byte
		part1 = append(part1, chars[:4]...)
		p1, t1 := processInput(append(part1, chars[12:16]...))
		fmt.Printf("Part1: found cost of %d in %s\n", p1, t1)
		p2, t2 := processInput(chars)
		fmt.Printf("Part2: found cost of %d in %s\n", p2, t2)
	}

	fmt.Printf("Global execution time (incl. parsing): %s\n", time.Since(start))

}

// Build the initial state, prepare the stack and start processing it.
func processInput(chars []byte) (uint32, time.Duration) {

	start := time.Now()

	m = [][4]uint8{}

	for i, v := range chars {
		if i%4 == 0 {
			m = append(m, [4]uint8{})
		}
		m[len(m)-1][i%4] = v - 65
	}

	s := State{
		h:         [11]uint8{free, free, free, free, free, free, free, free, free, free, free},
		remaining: uint8(len(chars)),
	}

	for r := len(m) - 1; r >= 0; r-- {
		for c, v := range m[r] {
			if int(v) == c && s.mobile[c] == 0 {
				s.remaining -= 1
			} else {
				s.mobile[c] += 1
			}
		}
	}

	target = 0
	stack = []State{s}
	visited = make(map[Key]uint32)

	for len(stack) > 0 {
		n := len(stack) - 1
		s = stack[n]
		stack = stack[:n]
		run(&s)
	}

	return target, time.Since(start)

}

// Process state. The general pattern is to perform all possible hall->room and room->room
// moves in-place (swapping not implemented) then start stacking new states for room-hallway
// moves.
func run(s *State) {

	done := false

	// Restart the search until we can't perform any moves (a move might enable other ones).
	for !done {

		done = true

		// In-place hallway->room
		for _, src := range spots {
			v := s.h[src]
			if v != free {
				dst := (v + 1) * 2
				if s.mobile[v] == 0 && s.pathClear(src, dst) {
					s.cost += uint32((absDiff(dst, src) + s.empty[v])) * weights[v]
					if s.cost >= target && target != 0 {
						return
					}
					if s.remaining == 1 {
						target = s.cost
						return
					}
					s.h[src] = free
					s.empty[v]--
					s.remaining--
					done = false
				}
			}
		}

		// In-place room->room
		for c := uint8(0); c < 4; c++ {
			if s.mobile[c] > 0 {
				r := s.empty[c]
				v := m[r][c]
				src := (c + 1) * 2
				dst := (v + 1) * 2
				if s.mobile[v] == 0 && s.pathClear(src, dst) {
					s.cost += uint32((absDiff(dst, src))+r+1+s.empty[v]) * weights[v]
					if s.cost >= target && target != 0 {
						return
					}
					s.empty[c]++
					s.mobile[c]--
					s.empty[v]--
					s.remaining--
					done = false
				}
			}
		}

	}

	// Enable with -e flag. Trying to discard a few states without adding complexity.
	// Evaluating a very approximate minimum remaining cost using each room theorical "best case"
	// remaining cost. The value is added to the cost when challenging the current target
	// in the next block.
	var min uint32
	if experimental && target != 0 {
		min = minCosts[0][s.mobile[0]+s.empty[0]] +
			minCosts[1][s.mobile[1]+s.empty[1]] +
			minCosts[2][s.mobile[2]+s.empty[2]] +
			minCosts[3][s.mobile[3]+s.empty[3]]
	}

	// New states (room->hallway)
	for c := uint8(0); c < 4; c++ {
		if s.mobile[c] > 0 {
			r := s.empty[c]
			src := (c + 1) * 2
			for _, dst := range spots {
				if s.h[dst] == free && s.pathClear(src, dst) {
					v := m[r][c]
					cost := s.cost + (uint32(absDiff(dst, src)+r+1) * weights[v])
					// Enable with -e flag. Trying to discard a few states without adding
					// complexity. As we take the (best-case) cost to fill the
					// room into account, we evaluate how much it would cost to move the amphipod
					// next to the room (closest hallway spot).
					var forecast uint32
					if experimental {
						forecast = (uint32(absDiff(dst, (v+1)*2)) - 1) * weights[v]
					}
					if cost+min+forecast < target || target == 0 {
						next := *s
						next.h[dst] = v
						next.empty[c]++
						next.mobile[c]--
						next.cost = cost
						key := Key{h: next.h, empty: next.empty, mobile: next.mobile}
						if value, ok := visited[key]; !ok || cost < value {
							stack = append(stack, next)
							visited[key] = cost
						}
					}
				}
			}
		}
	}

}

func absDiff(a uint8, b uint8) uint8 {
	if a < b {
		return b - a
	}
	return a - b
}
