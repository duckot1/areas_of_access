package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	lambert "github.com/duckot1/lambert_w_function"
)

type Player struct {
	x    float64
	y    float64
	maxV float64
	maxA float64
	velX float64
	velY float64
	id   int
}

type Result struct {
	yVal     int
	time     float64
	playerId int
}

type Group struct {
	xVal    int
	yVals   []int
	results []Result
}

var xLen = 100
var yLen = 70
var scaleFactor = 1
var numWorkersX = runtime.NumCPU()

// var numWorkersY = 1
// var numWorkersP = 1
// var fanPlayers = false
// var deepFan = false

var players = make([]Player, 30)

func main() {
	fmt.Println(runtime.NumCPU())
	for i := range players {
		players[i] = Player{
			x:    rand.Float64()*100 - 50,
			y:    rand.Float64() * 70,
			velX: rand.Float64() * 5,
			velY: rand.Float64() * 5,
			maxA: rand.Float64()*4 + 6,
			maxV: rand.Float64()*4 + 6,
			id:   i,
		}
	}

	calcAreas()
}

func calcAreas() {

	start := time.Now()

	groups := make([]Group, xLen*scaleFactor)
	in := gen(groups)

	// Distribute the work across a bunch of workers

	workers := make([]<-chan Group, numWorkersX)

	for i := range workers {
		workers[i] = doWork(in)
	}

	c := merge(workers)

	pitch := make([][]Result, xLen*scaleFactor)

	for group := range c {
		// fmt.Println(group.results)
		pitch[group.xVal] = group.results
	}

	// fmt.Println(pitch)

	fmt.Printf("Took %fs to ship %d packages\n", time.Since(start).Seconds()*1000, 100)
}

func gen(groups []Group) <-chan Group {
	out := make(chan Group)

	go func() {
		for i, group := range groups {
			group.xVal = i
			out <- group
		}
		close(out)
	}()

	return out
}

func genYChannel(yVals []int) <-chan int {
	out := make(chan int)

	go func() {
		for i := range yVals {
			yVals[i] = i
			out <- yVals[i]
		}
		close(out)
	}()

	return out
}

func genPChannel(players []Player) <-chan Player {
	out := make(chan Player)

	go func() {
		for _, player := range players {
			out <- player
		}
		close(out)
	}()

	return out
}

func doWork(in <-chan Group) <-chan Group {
	out := make(chan Group)

	go func() {
		for group := range in {

			group.yVals = make([]int, yLen*scaleFactor)
			group.results = make([]Result, yLen*scaleFactor)

			for i := range group.yVals {
				group.yVals[i] = i
				topResult := Result{yVal: i}

				for _, player := range players {
					time := getTimeToPoint(group.xVal, i, player)
					// fmt.Println(time, topResult.time)
					if time < topResult.time || topResult.time == 0.0 {
						topResult.time = time
						topResult.playerId = player.id
					}

				}

				group.results[i] = topResult
			}

			out <- group
		}
		close(out)
	}()

	return out
}

func getTimeToPoint(x int, y int, player Player) float64 {

	var pointA = []float64{float64(x), float64(y)}
	var pointB = []float64{player.x, player.y}

	d := getDistance(pointA, pointB)
	c := getInitialvelocityToPoint(pointA, player)

	b := player.maxV
	a := player.maxA

	pow := (a * (-(math.Pow(b, 2) / a) + (c*b)/a - d)) / math.Pow(b, 2)
	lam := lambert.GslSfLambertW0E(-((b - c) * math.Pow(math.E, pow) * math.Log(math.E)) / b).Val
	top := (math.Pow(b, 2) * lam) + (a * d * math.Log(math.E)) + (math.Pow(b, 2) * math.Log(math.E)) - (b * c * math.Log(math.E))
	time := top / (a * b * math.Log(math.E))

	return time
}

func merge(workers []<-chan Group) <-chan Group {
	var wg sync.WaitGroup
	out := make(chan Group)

	output := func(c <-chan Group) {
		for group := range c {
			out <- group
		}
		wg.Done()
	}

	wg.Add(len(workers))

	for _, worker := range workers {
		go output(worker)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func getDistance(pointA []float64, pointB []float64) float64 {
	pointA[0] = pointA[0]/float64(scaleFactor) - float64(xLen)/2
	pointA[1] = pointA[1] / float64(scaleFactor)
	x := math.Abs(pointA[0] - pointB[0])
	y := math.Abs(pointA[1] - pointB[1])

	distance := math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2))

	return distance
}

func getInitialvelocityToPoint(point []float64, player Player) float64 {

	getVel := func(d float64, v float64, angle float64) float64 {
		sign := -1.0
		if (v < 0 && d < 0) || (v > 0 && d > 0) {
			sign = 1.0
		}
		vel := math.Abs(v*math.Cos(angle)) * sign
		return vel
	}

	dy := point[1] - player.y
	dx := point[0] - player.x

	angleY := math.Atan(dx / dy)
	velY := getVel(dy, player.velY, angleY)

	angleX := math.Atan(dy / dx)
	velX := getVel(dx, player.velX, angleX)

	speed := velY + velX

	return speed
}

// func getClosestPlayer(in <-chan Player, x int, y int) <-chan Result {
// 	out := make(chan Result)
//
// 	go func() {
// 		for player := range in {
// 			time := getTimeToPoint(x, y, player)
// 			result := Result{yVal: y, time: time, playerId: player.id}
// 			out <- result
// 		}
// 		close(out)
// 	}()
//
// 	return out
// }

// func doWork(in <-chan Group) <-chan Group {
// 	out := make(chan Group)
//
// 	go func() {
// 		for group := range in {
//
// 			group.yVals = make([]int, yLen*scaleFactor)
// 			group.results = make([]Result, yLen*scaleFactor)
// 			in := genYChannel(group.yVals)
//
// 			workers := make([]<-chan Result, numWorkersY)
//
// 			for i := range workers {
// 				workers[i] = getResult(in, group.xVal)
// 			}
// 			c := mergeResults(workers)
//
// 			for res := range c {
// 				group.results[res.yVal] = res
// 			}
// 			out <- group
// 		}
// 		close(out)
// 	}()
//
// 	return out
// }
//
// func getResult(in <-chan int, x int) <-chan Result {
// 	out := make(chan Result)
// 	go func() {
// 		for yVal := range in {
//
// 			// Without channels
// 			if !fanPlayers {
// 				topResult := Result{yVal: yVal}
//
// 				for _, player := range players {
// 					time := getTimeToPoint(x, yVal, player)
// 					// fmt.Println(time, topResult.time)
// 					if time < topResult.time || topResult.time == 0.0 {
// 						topResult.time = time
// 						topResult.playerId = player.id
// 					}
//
// 				}
// 				out <- topResult
// 			} else {
// 				//With channels
//
// 				var topResult Result
//
// 				in := genPChannel(players)
//
// 				workers := make([]<-chan Result, numWorkersP)
//
// 				for i := range workers {
// 					workers[i] = getClosestPlayer(in, x, yVal)
// 				}
//
// 				c := mergeTime(workers)
//
// 				for res := range c {
// 					if res.time < topResult.time || topResult.time == 0.0 {
// 						topResult = res
// 					}
// 				}
// 				out <- topResult
// 			}
// 		}
// 		close(out)
// 	}()
// 	return out
// }

// func mergeTime(workers []<-chan Result) <-chan Result {
// 	var wg sync.WaitGroup
// 	out := make(chan Result)
//
// 	output := func(c <-chan Result) {
// 		for result := range c {
// 			out <- result
// 		}
// 		wg.Done()
// 	}
//
// 	wg.Add(len(workers))
//
// 	for _, worker := range workers {
// 		go output(worker)
// 	}
//
// 	go func() {
// 		wg.Wait()
// 		close(out)
// 	}()
//
// 	return out
// }
//
// func mergeResults(workers []<-chan Result) <-chan Result {
// 	var wg sync.WaitGroup
// 	out := make(chan Result)
//
// 	output := func(c <-chan Result) {
// 		for result := range c {
// 			out <- result
// 		}
// 		wg.Done()
// 	}
//
// 	wg.Add(len(workers))
//
// 	for _, worker := range workers {
// 		go output(worker)
// 	}
//
// 	go func() {
// 		wg.Wait()
// 		close(out)
// 	}()
//
// 	return out
// }
