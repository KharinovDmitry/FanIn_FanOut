package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Interval struct {
	start int
	end   int
}

type Input struct {
	FilePath  string
	Timeout   int
	Intervals []Interval
}

type PrimesList struct {
	interval Interval
	primes   []int
}

func main() {
	input := ParseFlags()
	file, err := os.Create(input.FilePath)
	if err != nil {
		log.Fatal(err)
	}

	wgFanOut := sync.WaitGroup{}
	wgFanOut.Add(len(input.Intervals))
	primeListChan := make(chan PrimesList, len(input.Intervals))
	for _, interval := range input.Intervals {
		ctx, _ := context.WithTimeout(context.Background(), time.Duration(input.Timeout)*time.Second)
		go func() {
			defer wgFanOut.Done()
			SearchPrime(ctx, interval, primeListChan)
		}()
	}

	go func() {
		wgFanOut.Wait()
		close(primeListChan)
	}()

	wgFanIn := sync.WaitGroup{}
	wgFanIn.Add(1)
	go func() {
		defer wgFanIn.Done()
		WriteRes(file, primeListChan)
	}()
	wgFanIn.Wait()
}

func SearchPrime(ctx context.Context, interval Interval, output chan<- PrimesList) {
	res := make([]int, 0)
	for i := interval.start; i < interval.end; i++ {
		select {
		case <-ctx.Done():
			break
		default:
			if IsPrime(i) {
				res = append(res, i)
			}
		}
	}

	output <- PrimesList{interval, res}
}

func WriteRes(file *os.File, results <-chan PrimesList) {
	writer := bufio.NewWriter(file)

	for res := range results {
		interval := fmt.Sprintf("%d:%d", res.interval.start, res.interval.end)
		_, err := fmt.Fprintln(writer, interval)
		if err != nil {
			log.Fatal(err)
		}
		for _, prime := range res.primes {
			_, err = fmt.Fprintln(writer, prime)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	err := writer.Flush()
	if err != nil {
		log.Fatal(err)
	}
}

func IsPrime(number int) bool {
	if number <= 1 {
		return false
	}

	for i := 2; i*i <= number; i++ {
		if number%i == 0 {
			return false
		}
	}
	return true
}

func ParseFlags() Input {
	res := Input{}
	for i := 1; i < len(os.Args); i += 2 {
		switch os.Args[i] {
		case "--file":
			res.FilePath = os.Args[i+1]
		case "--timeout":
			timeout, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				log.Fatal("Invalid value for --timeout")
				return Input{}
			}
			res.Timeout = timeout
		case "--range":
			bounds := strings.Split(os.Args[i+1], ":")
			if len(bounds) != 2 {
				log.Fatal("Invalid range format:", os.Args[i+1])
				return Input{}
			}
			start, err := strconv.Atoi(bounds[0])
			if err != nil {
				log.Fatal("Invalid range start:", bounds[0])
			}
			end, err := strconv.Atoi(bounds[1])
			if err != nil {
				log.Fatal("Invalid range end:", bounds[1])
			}
			res.Intervals = append(res.Intervals, Interval{start, end})
		}
	}
	return res
}
