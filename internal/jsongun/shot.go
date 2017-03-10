package jsongun

import (
	"sync"
	"context"
	"net/url"
	"time"
	"net/http"
	"io"
	"io/ioutil"
	"crypto/tls"
	"bytes"
)

type Shot struct {
	Uri     *url.URL
	Method  string
	Headers map[string]string
	Client  *http.Client
}

func NewShot(uri string) (target *Shot, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		target = nil
		return
	}

	tCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: 256,
	}
	c := &http.Client{
		Timeout: 15 * time.Second,
		Transport: tCfg,
	}

	target = Shot{
		Uri: u,
		Method: "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Client: c, }
	return
}

func (t *Shot) request(body string) (met Metrics, err error) {
	start := time.Now()

	// make request
	reqCtx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequest(t.Method, t.Uri.RequestURI(), bytes.NewBufferString(body))
	if err != nil {
		met = Metrics{IsError: true}
		return
	}
	for k, v := range t.Headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(reqCtx)

	// do it!
	ch := make(chan int)
	go func() {
		res, err := t.Client.Do(req)
		if err != nil {
			ch <- 0
			return
		}
		defer func() {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}()
		ch <- res.StatusCode
	}()

	// waiting...
	select {
	case <-ctx.Done():
		cancel()
		return
	case code := <-ch:
		if code == 0 {
			metCh <- Metrics{IsError: true}
			return
		}
		elapsed := time.Since(start)
		metCh <- Metrics{
			StatusCode: code,
			IsError: false,
			Latency: elapsed.Nanoseconds() / (1000 * 1000),
		}
		return
	}
}

func (t *Shot) Shoot(wg *sync.WaitGroup, jsonCh chan string, metCh chan Metrics, logCh chan string) {
	defer wg.Done()
	for body := range jsonCh {
		rCtx, cancel := context.WithCancel(context.Background())
		ch := make(chan Metrics)
		go t.request(rCtx, body, ch)

		select {
		case met := <-ch:
			metCh <- met
			close(ch)
		}
	}
}
