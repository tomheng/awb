package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
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
	version     = flag.Bool("v", false, "Print version number and exit")
)

const (
	CN      = "Another Web Bench"
	SN      = "awb"
	VERSION = "0.1"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s \n", SN)
	flag.PrintDefaults()
	os.Exit(1)
}

func showVersion() {
	fmt.Printf("%s version %s (%s distribution)\n", SN, VERSION, runtime.GOOS)
}

func showError(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *version {
		showVersion()
		os.Exit(0)
	}
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
