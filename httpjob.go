package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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
func (hj *httpJob) enableKeepAlive() {
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
