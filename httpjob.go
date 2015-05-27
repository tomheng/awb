package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type httpJob struct {
	timeout int64
	method  string
	url     string
	Header  http.Header
	Cookie  *http.Cookie
	data    url.Values
}

//HTTP request job
func newHttpJob(URL string, timeout int64, data, cookie, header string, keepAlive bool) *httpJob {
	hj := &httpJob{
		timeout,
		"GET",
		URL,
		make(http.Header),
		new(http.Cookie),
		url.Values{},
	}
	hj.Header.Set("User-Agent", SN+"/"+VERSION+" ("+CN+")")
	if len(data) > 0 {
		hj.addPostData(data)
	}
	if len(cookie) > 0 {
		hj.addCookie(cookie)
	}
	if len(header) > 0 {
		hj.addHeader(header)
	}
	if keepAlive {
		hj.enableKeepAlive()
	}
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
func (hj *httpJob) enableKeepAlive() {
	hj.Header.Add("Connection", "Keep-Alive")
}

//do HTTP request
func (hj *httpJob) perform() jobResulter {
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
	return httpJobResult{
		spendTime,
		response,
	}
}

type httpJobResult struct {
	spendTime time.Duration
	response  *http.Response
}

func (hjr httpJobResult) isSuccess() bool {
	if hjr.response != nil && hjr.response.StatusCode == 200 {
		return true
	}
	return false
}

func (hjr httpJobResult) getTotalLength() int64 {
	if !hjr.isSuccess() {
		return 0
	}
	res, _ := httputil.DumpResponse(hjr.response, true)
	return int64(len(res))
}

func (hjr httpJobResult) getContentLength() int64 {
	if !hjr.isSuccess() {
		return 0
	}
	if hjr.response.ContentLength == -1 {
		body, _ := ioutil.ReadAll(hjr.response.Body)
		hjr.response.ContentLength = int64(len(body))
	}
	return hjr.response.ContentLength
}

var once sync.Once

//print some help info
func (hjr httpJobResult) println() {
	once.Do(func() {
		fmt.Println("\nproto\tcode\ttotal_bytes\tbody_byte\ttime")
	})
	fmt.Printf("%s\t%d\t%v\t%v\t%.4fms\n", hjr.response.Proto, hjr.response.StatusCode, hjr.getContentLength(), hjr.getTotalLength(), hjr.spendTime.Seconds())
}
