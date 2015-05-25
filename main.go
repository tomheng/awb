package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	requests    = flag.Int("n", 100, "Number of requests to perform")
	concurrency = flag.Int("c", 10, "Number of multiple requests to make at a time")
	timelimit   = flag.Int("t", 0, "Seconds to max. to spend on benchmarking")
	timeout     = flag.Int("s", 30000, "Millisecond to max. wait for each response Default is 30 seconds")
)

const (
	CN = "Another Web Bench"
	SN = "awb"
	VERSION = "0.1"
)

type jober interface {
	perform() *jobResult
}

type bench struct {
	Requests    int
	Concurrency int
	Timelimit   int
	Jobs        chan jober
	Br          *benchResult
}

type benchResult struct {
	Spend_time        time.Duration
	Total_transferred int64
	Html_transferred  int64
	Success_count     int64
	Failed_count      int64
}

func newBench(r, c, t int) *bench {
	return &bench{
		Requests:    r,
		Concurrency: c,
		Timelimit:   t,
		Jobs:        make(chan jober),
		Br:          &benchResult{},
	}
}

func (b *bench) start(jobs ...jober) {
	var wg sync.WaitGroup
	go func() {
		if b.Timelimit <= 0 {
			return
		}
		time.Sleep(time.Second * time.Duration(b.Timelimit))
		b.stop()
	}()
	fmt.Printf("This is %s(%s), Version %s \n\n", SN, CN, VERSION)
	fmt.Println("start Benchmarking ...(be patient)")
	for _, job := range jobs {
		go b.produce(job)
	}

	for i := 0; i < b.Concurrency; i++ {
		wg.Add(1)
		go b.consume(&wg)
	}
	wg.Wait()
	b.printResult()
}

func (b *bench) printResult() {
	template_text := `
Concurrency Level:      %d
Time taken for tests:   %s
Complete requests:      %d
Failed requests:        %d

Total transferred:      %d bytes
HTML transferred:       %d bytes
Requests per second:    %.2f [#/sec] (mean)
Transfer rate:          %.2f [Kbytes/sec] received
`
	complete_request := b.Br.Success_count + b.Br.Failed_count
	fmt.Printf(template_text,
		b.Concurrency,
		b.Br.Spend_time,
		complete_request,
		b.Br.Failed_count,
		b.Br.Total_transferred,
		b.Br.Html_transferred,
		float64(complete_request) / b.Br.Spend_time.Seconds(),
		float64(b.Br.Total_transferred) / 1024 / b.Br.Spend_time.Seconds(),
	)
}

func (b *bench) stop() {
	close(b.Jobs)
}

func (b *bench) produce(job jober) {
	i := 0
	for {
		if i >= b.Requests {
			b.stop()
			break
		}
		i += 1
		b.Jobs <- job
	}
}

func (b *bench) consume(wg *sync.WaitGroup) {
	for job := range b.Jobs {
		result := job.perform()
		if result.success {
			b.Br.Success_count += 1
		} else {
			b.Br.Failed_count += 1
		}
		b.Br.Html_transferred += result.content_length
		b.Br.Total_transferred += result.total_length
		b.Br.Spend_time += result.spend_time
	}
	wg.Done()
}

type httpJob struct {
	client  *http.Client
	request *http.Request
}

func newHttpJob(timeout int64, method string, url string) *httpJob {
	hr, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return &httpJob{
		&http.Client{
			Timeout: time.Millisecond * time.Duration(timeout),
		},
		hr,
	}
}

func (hj *httpJob) perform() *jobResult {
	var content_length, total_length int64
	var content_type string
	start := time.Now()
	response, _ := hj.client.Do(hj.request)
	spend_time := time.Since(start)
	success := false
	if response.StatusCode == 200 {
		success = true
		if response.ContentLength == -1 {
			body, _ := ioutil.ReadAll(response.Body)
			response.ContentLength = int64(len(body))
		}
		res, _ := httputil.DumpResponse(response, true)
		total_length = int64(len(res))
		content_length = response.ContentLength
		content_type = response.Header.Get("Content-Type")
	}
	return &jobResult{
		success,
		spend_time,
		content_type,
		content_length,
		total_length,
	}
}

type jobResult struct {
	success        bool
	spend_time     time.Duration
	content_type   string
	content_length int64
	total_length   int64
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	URL := flag.Arg(0)
	b := newBench(*requests, *concurrency, *timelimit)
	job := newHttpJob(0, "GET", URL)
	b.start(job)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s \n", SN)
	flag.PrintDefaults()
	os.Exit(1)
}
