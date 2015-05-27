# awb
[![GoDoc](http://godoc.org/github.com/tomheng/awb?status.svg)](http://godoc.org/github.com/tomheng/awb) [![Build Status](https://travis-ci.org/tomheng/awb.svg)](https://travis-ci.org/tomheng/awb)

**awb** is another web bench.

## Install 

After installing Go and setting up your [GOPATH](http://golang.org/doc/code.html#GOPATH), then install the **awb** package (**go 1.1** or greater is required):
~~~
go get github.com/tomheng/awb
~~~

or you can download binary distribution, unzip and copy it to you PATH.

* [Linux 64-bit](http://blog.webfuns.net/awb-linux-64.zip)
* [OSX 64-bit](http://blog.webfuns.net/awb-osx-64.zip)

## Start bench

it is very simple using awb to bench a HTTP interface.

~~~
awb -n 10000 -n 100 http://localhost/bench
~~~

## Features

* more fast and concurrently
* HTTP and HTTPS
* support multi HTTP method(GET,POST et)
* more control on bench request
