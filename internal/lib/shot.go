package lib

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
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
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: 256,
	}
	c := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tCfg,
	}

	target = &Shot{
		Uri:     u,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Client:  c}
	return
}

func (t *Shot) request(body string) (met Metrics, err error) {
	start := time.Now()
	met = Metrics{IsError: true}

	// make request
	req, err := http.NewRequest(t.Method, t.Uri.String(), bytes.NewBufferString(body))
	if err != nil {
		return
	}
	for k, v := range t.Headers {
		req.Header.Add(k, v)
	}
	// req.Cancel = cancel

	// do it!
	res, err := t.Client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}()

	elapsed := time.Since(start)
	met = Metrics{
		StatusCode: strconv.Itoa(res.StatusCode),
		IsError:    false,
		Latency:    float64(elapsed.Nanoseconds()) / float64(1000*1000)}
	return
}

func (t *Shot) Shoot(wg *sync.WaitGroup, jsonCh chan string, metCh chan Metrics, logCh chan string) {
	defer wg.Done()
	for body := range jsonCh {
		met, err := t.request(body)
		if err != nil {
			logCh <- fmt.Sprintf("ERROR: %s", err)
		}
		metCh <- met
	}
}
