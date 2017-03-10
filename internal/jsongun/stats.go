package jsongun

import (
	"context"
	"sync"
	"time"
	"github.com/Sirupsen/logrus/hooks/syslog"
)

type Stats struct {
	beginTime      string
	endTime        string
	durationSec    float32
	totalCount     int32
	timeoutCount   int32
	status2xxCount int32
	status3xxCount int32
	status4xxCount int32
	status5xxCount int32
	reqLatencyMin  float32
	reqLatencyP10  float32
	reqLatencyP50  float32
	reqLatencyP90  float32
	reqLatencyP99  float32
	reqLatencyMax  float32
	reqLatencyAve  float32
}

type Accum struct {
	metrics []Metrics
	logs    []string
	mux     sync.RWMutex
}

type Result struct {
	total  Stats
	window Stats
	mux    sync.RWMutex
}

func CalcOutputStats(ctx context.Context, wg *sync.WaitGroup, metCh chan Metrics, logCh chan string) {
	defer wg.Done()

	a := &Accum{}
	r := &Result{}
	r.total.beginTime = time.Now().UTC().Format(time.RFC3339)

	localCtx, cancel := context.WithCancel(context.Background())
	var localWg sync.WaitGroup
	localWg.Add(2)
	done := make(chan struct{})
	go func() {
		localWg.Wait()
		done <- true
	}()
	go accumulateMetrics(localCtx, &localWg, metCh, a)
	go accumulateLogs(localCtx, &localWg, logCh, a)

	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				calculateStats(a, r)
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			cancel()
			close(quit)
			return
		case <-done:
			close(quit)
			return
		}
	}
}

func accumulateMetrics(ctx context.Context, wg *sync.WaitGroup, metCh chan Metrics, a *Accum) {
	defer wg.Done()
	for{
		select{
		case <-ctx.Done():
		case met, ok := metCh:
			if !ok {

			}
		}
	}
}

func accumulateLogs(ctx context.Context, wg *sync.WaitGroup, logCh chan string, a *Accum) {

}

func tick(a *Accum, r *Result) {
	calculateStats(a, r)
	printLogo()
	printTotal(r)
	printWindow(r)
	printLogs(a)
}

func calculateStats(a *Accum, r *Result) {

}

func printLogo() {

}

func printWindow(r *Result) {
}

func printTotal(r *Result) {
}

func printLogs(a *Accum) {
}