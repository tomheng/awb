package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	requests    = flag.Int64("n", 100, "Number of requests to perform")
	concurrency = flag.Int64("c", 10, "Number of multiple requests to make at a time")
	timelimit   = flag.Int64("t", 0, "Seconds to max. to spend on benchmarking")
	timeout     = flag.Int64("s", 30, "Seconds to max. wait for each response Default is 30 seconds")
)

type jober interface {
	perform()
}

type bench struct {
	requests    int
	concurrency int
	timelimit   int
	jobs        chan jober
}

func newBench(r, c, t int64) *bench {
	return &bench{
		requests:    r,
		concurrency: c,
		timelimit:   t,
	}
}

func (b *bench) start(jobs ...Jober) {
	var wg sync.WaitGroup
	go func() {
		time.Sleep(time.Duration * b.timelimit)
		b.stop()
	}()
	for job := range jobs {
		go b.produce(job)
	}

	for i := 0; i < b.concurrency; i++ {
		wg.Add(1)
		go b.consume(&wg)
	}
	wg.Wait()
	b.printResult()
}

func (b *bench) printResult() {

}

func (b *bench) stop() {
	close(b.jobs)
}

func (b *bench) produce(Jober job) {
	i := 0
	for {
		if i >= b.requests {
			b.stop()
			break
		}
		i += 1
		b.jober <- job
	}
}

func (b *bench) consume(wg *sync.WaitGroup) {
	for job := range b.jobs {
		go job.perform()
	}
	wg.Done()
}

type httpJob struct {
	client  *http.Client
	request *http.Request
}

func newHttpJob(timeout int64, method string, url string) {
	return &httpJob{
		&http.Client{
			Timeout: time.Duration(time.Millisecond * timeout),
		},
		http.NewRequest(method, url, nil),
	}
}

func (hj *httpJob) perform() {
	response, _ := hr.client.Do(hr.reqest)
	if response.StatusCode == 200 {
		response.ContentLength;
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)
	}
}

func main() {
	flag.Usage = usage
	if flag.NArg() != 1 {
		usage()
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	URL := flag.Arg(0)
	b := newBench(requests, concurrency, timelimit)
	b.start(job)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}
