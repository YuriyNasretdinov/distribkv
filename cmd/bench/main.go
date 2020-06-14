package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	addr           = flag.String("addr", "localhost:8080", "The HTTP host port for the instance that is benchmarked.")
	iterations     = flag.Int("iterations", 1000, "The number of iterations for writing")
	readIterations = flag.Int("read-iterations", 100000, "The number of iterations for reading")
	concurrency    = flag.Int("concurrency", 1, "How many goroutines to run in parallel when doing writes")
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		IdleConnTimeout:     time.Second * 60,
		MaxIdleConns:        300,
		MaxConnsPerHost:     300,
		MaxIdleConnsPerHost: 300,
	},
}

func benchmark(name string, iter int, fn func() string) (qps float64, strs []string) {
	var max time.Duration
	var min = time.Hour

	start := time.Now()
	for i := 0; i < iter; i++ {
		iterStart := time.Now()
		strs = append(strs, fn())
		iterTime := time.Since(iterStart)
		if iterTime > max {
			max = iterTime
		}
		if iterTime < min {
			min = iterTime
		}
	}

	avg := time.Since(start) / time.Duration(iter)
	qps = float64(iter) / (float64(time.Since(start)) / float64(time.Second))
	fmt.Printf("Func %s took %s avg, %.1f QPS, %s max, %s min\n", name, avg, qps, max, min)

	return qps, strs
}

func writeRand() (key string) {
	key = fmt.Sprintf("key-%d", rand.Intn(1000000))
	value := fmt.Sprintf("value-%d", rand.Intn(1000000))

	values := url.Values{}
	values.Set("key", key)
	values.Set("value", value)

	resp, err := httpClient.Get("http://" + (*addr) + "/set?" + values.Encode())
	if err != nil {
		log.Fatalf("Error during set: %v", err)
	}

	io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()

	return key
}

func readRand(allKeys []string) (key string) {
	key = allKeys[rand.Intn(len(allKeys))]

	values := url.Values{}
	values.Set("key", key)

	resp, err := httpClient.Get("http://" + (*addr) + "/get?" + values.Encode())
	if err != nil {
		log.Fatalf("Error during get: %v", err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()

	return key
}

func benchmarkWrite() (allKeys []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalQPS float64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			qps, strs := benchmark("write", *iterations, writeRand)
			mu.Lock()
			totalQPS += qps
			allKeys = append(allKeys, strs...)
			mu.Unlock()

			wg.Done()
		}()
	}

	wg.Wait()

	log.Printf("Write total QPS: %.1f, set %d keys", totalQPS, len(allKeys))

	return allKeys
}

func benchmarkRead(allKeys []string) {
	var totalQPS float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			qps, _ := benchmark("read", *readIterations, func() string { return readRand(allKeys) })
			mu.Lock()
			totalQPS += qps
			mu.Unlock()

			wg.Done()
		}()
	}

	wg.Wait()

	log.Printf("Read total QPS: %.1f", totalQPS)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	fmt.Printf("Running with %d iterations and concurrency level %d\n", *iterations, *concurrency)

	allKeys := benchmarkWrite()

	go benchmarkWrite()
	benchmarkRead(allKeys)
}
