package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	requests    = flag.Int("n", 0, "Number of requests to perform")
	concurrency = flag.Int("c", 10, "Number of multiple requests to make at a time")
	timelimit   = flag.Int("t", 0, "Seconds to max. to spend on benchmarking")
	timeout     = flag.Int64("s", 30000, "Millisecond to max. wait for each response Default is 30 seconds")
	data        = flag.String("d", "", "(HTTP) Sends the specified data in a POST request to the HTTP server")
	cookie      = flag.String("b", "", "Pass  the  data  to the HTTP server as a cookie")
	header      = flag.String("H", "", "(HTTP) Extra header to include in the request when sending HTTP to a server")
	keepAlive   = flag.Bool("k", false, "Use HTTP KeepAlive feature")
)

const (
	CN      = "Another Web Bench"
	SN      = "awb"
	VERSION = "0.1"
)

type jober interface {
	perform() jobResult
}

type bench struct {
	Requests    int
	Concurrency int
	Timelimit   int
	Jobs        chan jober
	Br          *benchResult
	stoped      bool
	sync.Mutex
}

type benchResult struct {
	SpendTime        time.Duration
	TotalTransferred int64
	HtmlTransferred  int64
	SuccessCount     int64
	FailedCount      int64
}

func newBench(r, c, t int) *bench {
	return &bench{
		Requests:    r,
		Concurrency: c,
		Timelimit:   t,
		Jobs:        make(chan jober, c*2),
		Br:          &benchResult{},
		stoped:      false,
	}
}

func (b *bench) start(jobs ...jober) {
	var wg sync.WaitGroup
	//set timer if with timelimit
	go func() {
		if b.Timelimit <= 0 {
			return
		}
		time.Sleep(time.Second * time.Duration(b.Timelimit))
		b.stop()
	}()
	fmt.Printf("This is %s(%s), Version %s \n\n", SN, CN, VERSION)
	fmt.Println("start Benchmarking ...(be patient)")
	start := time.Now()
	for _, job := range jobs {
		go b.produce(job)
	}

	for i := 0; i < b.Concurrency; i++ {
		wg.Add(1)
		go b.consume(&wg)
	}
	wg.Wait()
	b.Br.SpendTime = time.Since(start)
	b.printResult()
}

func (b *bench) printResult() {
	templateText := `
Concurrency Level:      %d
Time taken for tests:   %s
Complete requests:      %d
Failed requests:        %d

Total transferred:      %d bytes
HTML transferred:       %d bytes
Requests per second:    %.2f [#/sec] (mean)
Transfer rate:          %.2f [Kbytes/sec] received
`
	completeRequest := b.Br.SuccessCount + b.Br.FailedCount
	fmt.Printf(templateText,
		b.Concurrency,
		b.Br.SpendTime,
		completeRequest,
		b.Br.FailedCount,
		b.Br.TotalTransferred,
		b.Br.HtmlTransferred,
		float64(completeRequest)/b.Br.SpendTime.Seconds(),
		float64(b.Br.TotalTransferred)/1024/b.Br.SpendTime.Seconds(),
	)
}

func (b *bench) stop() {
	b.Lock()
	defer b.Unlock()
	if b.stoped {
		return
	}
	b.stoped = true
	close(b.Jobs)
}

func (b *bench) processResult(result jobResult) {
	b.Lock()
	defer b.Unlock()
	if result.success {
		b.Br.SuccessCount += 1
	} else {
		b.Br.FailedCount += 1
	}
	b.Br.HtmlTransferred += result.contentLength
	b.Br.TotalTransferred += result.totalLength
}

func (b *bench) produce(job jober) {
	i := 0
	for {
		b.Lock()
		if b.stoped {
			b.Unlock()
			break
		}
		if b.Requests > 0 && i >= b.Requests {
			b.Unlock()
			b.stop()
			break
		}
		i += 1
		b.Jobs <- job
		b.Unlock()
	}
}

//concurrency unit
func (b *bench) consume(wg *sync.WaitGroup) {
	for job := range b.Jobs {
		//Asynchronous process job result
		result := job.perform()
		b.processResult(result)
	}
	wg.Done()
}

type httpJob struct {
	timeout int64
	method  string
	url     string
	Header  http.Header
	Cookie  *http.Cookie
	data    url.Values
}

//HTTP request job
func newHttpJob(URL string, timeout int64) *httpJob {
	hj := &httpJob{
		timeout,
		"GET",
		URL,
		make(http.Header),
		new(http.Cookie),
		url.Values{},
	}
	hj.Header.Set("User-Agent", SN+"/"+VERSION+" ("+CN+")")
	return hj
}

//add request header
func (hj *httpJob) addHeader(header string) {
	headerList := strings.Split(header, "\n")
	for _, h := range headerList {
		hh := strings.Split(h, "=")
		if len(hh) < 2 {
			continue
		}
		hj.Header.Add(hh[0], hh[1])
	}
}

//add request post data
func (hj *httpJob) addPostData(data string) {
	value, err := url.ParseQuery(data)
	if err == nil {
		hj.method = "POST"
		hj.data = value
	}
}

//add request cookie
func (hj *httpJob) addCookie(cookie string) {
	hj.Header.Set("Cookie", cookie)
}

//enable request keepAlive
func (hj *httpJob) enableKeepAlive(){
	hj.Header.Add("Connection", "Keep-Alive")
}

//do HTTP request
func (hj *httpJob) perform() jobResult {
	var contentLength, totalLength int64
	var contentType string
	client := http.Client{
		Timeout: time.Millisecond * time.Duration(hj.timeout),
	}
	hr, err := http.NewRequest(hj.method, hj.url, nil)
	if err != nil {
		log.Fatal("failed to initialize http request, please try again")
	}
	hr.Header = hj.Header
	start := time.Now()
	response, err := client.Do(hr)
	spendTime := time.Since(start)
	success := false
	if err == nil && response.StatusCode == 200 {
		success = true
		if response.ContentLength == -1 {
			body, _ := ioutil.ReadAll(response.Body)
			response.ContentLength = int64(len(body))
		}
		res, _ := httputil.DumpResponse(response, true)
		totalLength = int64(len(res))
		contentLength = response.ContentLength
		contentType = response.Header.Get("Content-Type")
	}
	return jobResult{
		success,
		spendTime,
		contentType,
		contentLength,
		totalLength,
	}
}

type jobResult struct {
	success       bool
	spendTime     time.Duration
	contentType   string
	contentLength int64
	totalLength   int64
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	URL := flag.Arg(0)
	if *requests < 1 && *timelimit < 1 {
		showError("benchmark request number or speed time must be set")
	}
	if *timelimit > 0 {
		*requests = 0
	}
	b := newBench(*requests, *concurrency, *timelimit)
	job := newHttpJob(URL, *timeout)
	if len(*data) > 0 {
		job.addPostData(*data)
	}
	if len(*cookie) > 0 {
		job.addCookie(*cookie)
	}
	if len(*header) > 0 {
		job.addHeader(*header)
	}
	if *keepAlive {
		job.enableKeepAlive()
	}
	//listen signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		// Block until a signal is received.
		<-c
		b.stop()
	}()
	b.start(job)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s \n", SN)
	flag.PrintDefaults()
	os.Exit(1)
}

func showError(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
