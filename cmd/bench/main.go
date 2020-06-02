package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	addr        = flag.String("addr", "localhost:8080", "The HTTP host port for the instance that is benchmarked.")
	iterations  = flag.Int("iterations", 1000, "The number of iterations for writing")
	concurrency = flag.Int("concurrency", 1, "How many goroutines to run in parallel when doing writes")
)

func benchmark(name string, fn func()) {
	var max time.Duration
	var min = time.Hour

	start := time.Now()
	for i := 0; i < *iterations; i++ {
		iterStart := time.Now()
		fn()
		iterTime := time.Since(iterStart)
		if iterTime > max {
			max = iterTime
		}
		if iterTime < min {
			min = iterTime
		}
	}

	avg := time.Since(start) / time.Duration(*iterations)
	qps := float64(*iterations) / (float64(time.Since(start)) / float64(time.Second))
	fmt.Printf("Func %s took %s avg, %.1f QPS, %s max, %s min\n", name, avg, qps, max, min)
}

func writeRand() {
	key := fmt.Sprintf("key-%d", rand.Intn(1000000))
	value := fmt.Sprintf("value-%d", rand.Intn(1000000))

	values := url.Values{}
	values.Set("key", key)
	values.Set("value", value)

	resp, err := http.Get("http://" + (*addr) + "/set?" + values.Encode())
	if err != nil {
		log.Fatalf("Error during set: %v", err)
	}
	defer resp.Body.Close()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	fmt.Printf("Running with %d iterations and concurrency level %d\n", *iterations, *concurrency)

	var wg sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			benchmark("write", writeRand)
			wg.Done()
		}()
	}

	wg.Wait()
}
