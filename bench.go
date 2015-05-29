package main

import (
	"fmt"
	"sync"
	"time"
)

//job interface
type jober interface {
	perform() jobResulter
}

//job result interface
type jobResulter interface {
	isSuccess() bool
	getTotalLength() int64
	getContentLength() int64
}

type bench struct {
	Requests    int
	Concurrency int
	Timelimit   int
	Jobs        chan jober
	Br          *benchResult
	stoped      chan bool
	verbose     bool
	sync.Mutex
}

type benchResult struct {
	SpendTime        time.Duration
	TotalTransferred int64
	HtmlTransferred  int64
	SuccessCount     int64
	FailedCount      int64
}

func newBench(r, c, t int, v bool) *bench {
	return &bench{
		Requests:    r,
		Concurrency: c,
		Timelimit:   t,
		Jobs:        make(chan jober, c*2),
		Br:          &benchResult{},
		stoped:      make(chan bool),
		verbose:     v,
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
	b.stoped <- true
}

//process bench result
func (b *bench) processResult(result jobResulter, wg *sync.WaitGroup) {
	defer wg.Done()
	b.Lock()
	defer b.Unlock()
	if result.isSuccess() {
		b.Br.SuccessCount += 1
	} else {
		b.Br.FailedCount += 1
	}
	b.Br.HtmlTransferred += result.getContentLength()
	b.Br.TotalTransferred += result.getTotalLength()
	if b.verbose {
		fmt.Println(result)
	}
}

//bench producer
func (b *bench) produce(job jober) {
	defer close(b.Jobs)
	i := 0
	for {
		select {
		case <-b.stoped:
			return
		default:
			if b.Requests > 0 && i >= b.Requests {
				return
			}
			i += 1
			b.Jobs <- job
		}
	}
}

//concurrency unit
func (b *bench) consume(wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range b.Jobs {
		//Asynchronous process job result
		result := job.perform()
		wg.Add(1)
		go b.processResult(result, wg)
	}
}
