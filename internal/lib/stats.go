package lib

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"fmt"

	"github.com/buger/goterm"
)

type Stats struct {
	beginTime      time.Time
	endTime        time.Time
	durationSec    float64
	successCount   uint64
	errorCount     uint64
	status2xxCount uint64
	status3xxCount uint64
	status4xxCount uint64
	status5xxCount uint64
	reqLatencyMin  float64
	reqLatencyP10  float64
	reqLatencyP50  float64
	reqLatencyP90  float64
	reqLatencyP99  float64
	reqLatencyMax  float64
	reqLatencyAve  float64
}

type Accum struct {
	totalCount     uint64
	successCount   uint64
	errorCount     uint64
	status2xxCount uint64
	status3xxCount uint64
	status4xxCount uint64
	status5xxCount uint64
	reqLatency     []float64
	logs           []string
	reqLatencyAll  []float64
	mux            sync.RWMutex
}

type Result struct {
	total  Stats
	window Stats
	mux    sync.RWMutex
}

func (a *Accum) reset() {
	a.totalCount = 0
	a.successCount = 0
	a.errorCount = 0
	a.status2xxCount = 0
	a.status3xxCount = 0
	a.status4xxCount = 0
	a.status5xxCount = 0
	a.reqLatency = nil
}

func CalcOutputStats(wg *sync.WaitGroup, metCh chan Metrics, logCh chan string) {
	defer wg.Done()

	a := &Accum{}
	r := &Result{}
	now := time.Now()
	preTickTime := &now
	r.total.beginTime = *preTickTime

	var localWg sync.WaitGroup
	localWg.Add(2)
	go accumulateMetrics(&localWg, metCh, a)
	go accumulateLogs(&localWg, logCh, a)
	done := make(chan bool)
	go func() {
		localWg.Wait()
		done <- true
	}()

	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				tick(a, r, preTickTime)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			quit <- true
			tick(a, r, preTickTime)
			return
		}
	}
}

func accumulateMetrics(wg *sync.WaitGroup, metCh chan Metrics, a *Accum) {
	defer wg.Done()
	for met := range metCh {
		a.mux.Lock()
		a.totalCount += 1
		if met.IsError {
			a.errorCount += 1
			a.mux.Unlock()
			continue
		}
		a.successCount += 1
		switch {
		case strings.HasPrefix(met.StatusCode, "2"):
			a.status2xxCount += 1
		case strings.HasPrefix(met.StatusCode, "3"):
			a.status3xxCount += 1
		case strings.HasPrefix(met.StatusCode, "4"):
			a.status4xxCount += 1
		case strings.HasPrefix(met.StatusCode, "5"):
			a.status5xxCount += 1
		}
		a.reqLatency = append(a.reqLatency, met.Latency)
		a.reqLatencyAll = append(a.reqLatencyAll, met.Latency)
		a.mux.Unlock()
	}
}

func accumulateLogs(wg *sync.WaitGroup, logCh chan string, a *Accum) {
	defer wg.Done()
	for log := range logCh {
		a.mux.Lock()
		isContained := false
		for _, l := range a.logs {
			if l == log {
				isContained = true
				break
			}
		}
		if !isContained {
			a.logs = append(a.logs, log)
		}
		a.mux.Unlock()
	}
}

func tick(a *Accum, r *Result, preTickTime *time.Time) {
	now := time.Now()
	calculateStats(*preTickTime, now, a, r)
	*preTickTime = now
	printReport(a, r)
}

func calculateStats(begin time.Time, end time.Time, a *Accum, r *Result) {
	if a.totalCount == 0 {
		return
	}
	a.mux.RLock()
	r.mux.Lock()
	defer a.mux.RUnlock()
	defer r.mux.Unlock()

	// init window
	r.window = Stats{}

	// calc window percentile
	if l := len(a.reqLatency); l > 0 {
		sort.Float64s(a.reqLatency)
		r.window.reqLatencyP10 = a.reqLatency[int(math.Ceil(float64(l)*0.1)-1)]
		r.window.reqLatencyP50 = a.reqLatency[int(math.Ceil(float64(l)*0.5)-1)]
		r.window.reqLatencyP90 = a.reqLatency[int(math.Ceil(float64(l)*0.9)-1)]
		r.window.reqLatencyP99 = a.reqLatency[int(math.Ceil(float64(l)*0.99)-1)]
		r.window.reqLatencyMin = a.reqLatency[0]
		r.window.reqLatencyMax = a.reqLatency[l-1]
		s := 0.0
		for _, v := range a.reqLatency {
			s += v
		}
		r.window.reqLatencyAve = s / float64(l)
	}
	// calc total percentile
	if l := len(a.reqLatencyAll); l > 0 {
		sort.Float64s(a.reqLatencyAll)
		r.total.reqLatencyP10 = a.reqLatencyAll[int(math.Ceil(float64(l)*0.1)-1)]
		r.total.reqLatencyP50 = a.reqLatencyAll[int(math.Ceil(float64(l)*0.5)-1)]
		r.total.reqLatencyP90 = a.reqLatencyAll[int(math.Ceil(float64(l)*0.9)-1)]
		r.total.reqLatencyP99 = a.reqLatencyAll[int(math.Ceil(float64(l)*0.99)-1)]
		r.total.reqLatencyMin = a.reqLatencyAll[0]
		r.total.reqLatencyMax = a.reqLatencyAll[l-1]
		s := 0.0
		for _, v := range a.reqLatencyAll {
			s += v
		}
		r.total.reqLatencyAve = s / float64(l)
	}

	// count
	r.window.successCount = a.successCount
	r.window.errorCount = a.errorCount
	r.window.status2xxCount = a.status2xxCount
	r.window.status3xxCount = a.status3xxCount
	r.window.status4xxCount = a.status4xxCount
	r.window.status5xxCount = a.status5xxCount
	r.total.successCount += a.successCount
	r.total.errorCount += a.errorCount
	r.total.status2xxCount += a.status2xxCount
	r.total.status3xxCount += a.status3xxCount
	r.total.status4xxCount += a.status4xxCount
	r.total.status5xxCount += a.status5xxCount

	// time
	r.window.beginTime = begin
	r.window.endTime = end
	r.window.durationSec = float64(end.UnixNano()-begin.UnixNano()) / float64(1000*1000*1000)
	r.total.endTime = end
	r.total.durationSec = float64(end.UnixNano()-r.total.beginTime.UnixNano()) / float64(1000*1000*1000)

	// init accum
	a.reset()
}

func printReport(a *Accum, r *Result) {
	goterm.Clear()
	goterm.MoveCursor(1, 1)
	printLogo()
	printTotal(r)
	printWindow(r)
	printLogs(a)
	goterm.Flush()
}

func printLogo() {
	goterm.Println(`    __ _      __
  |(_ / \|\ |/__| ||\ |
\_|__)\_/| \|\_||_|| \|
                        `)
}

func printStats(s Stats) {
	tables := goterm.NewTable(0, 15, 2, ' ', 0)
	columns := []string{
		"From",
		"To",
		"Duration[s]",
		"OK(/sec)",
		"NG",
		"2xx(R[%])",
		"3xx(R[%])",
		"4xx(R[%])",
		"5xx(R[%])",
		"Min[ms]",
		"P10[ms]",
		"P50[ms]",
		"P90[ms]",
		"P99[ms]",
		"Max[ms]",
		"Ave[ms]",
	}
	column := strings.Join(columns, "\t")
	fmt.Fprint(tables, column+"\n")
	fmt.Fprintf(tables,
		"%s\t%s\t%6.1f\t%s\t%d\t%s\t%s\t%s\t%s\t%6.3f\t%6.3f\t%6.3f\t%6.3f\t%6.3f\t%6.3f\t%6.3f\n",
		s.beginTime.Format("15:04:05"),
		s.endTime.Format("15:04:05"),
		s.durationSec,
		fmt.Sprintf("%d(%.2f/sec)", s.successCount, float64(s.successCount)/s.durationSec),
		s.errorCount,
		fmt.Sprintf("%d(%4.1f)", s.status2xxCount, float64(s.status2xxCount)/float64(s.successCount)*100),
		fmt.Sprintf("%d(%4.1f)", s.status3xxCount, float64(s.status3xxCount)/float64(s.successCount)*100),
		fmt.Sprintf("%d(%4.1f)", s.status4xxCount, float64(s.status4xxCount)/float64(s.successCount)*100),
		fmt.Sprintf("%d(%4.1f)", s.status5xxCount, float64(s.status5xxCount)/float64(s.successCount)*100),
		s.reqLatencyMin,
		s.reqLatencyP10,
		s.reqLatencyP50,
		s.reqLatencyP90,
		s.reqLatencyP99,
		s.reqLatencyMax,
		s.reqLatencyAve)
	goterm.Print(tables)
}

func printTotal(r *Result) {
	//box := goterm.NewBox(20, 3, 0)
	//fmt.Fprint(box, "TOTAL")
	//goterm.Println(box.String())
	goterm.Println("----TOTAL----")
	r.mux.RLock()
	defer r.mux.RUnlock()
	printStats(r.total)
}

func printWindow(r *Result) {
	//box := goterm.NewBox(20, 3, 0)
	//fmt.Fprint(box, "CURRENT")
	//goterm.Println(box.String())
	goterm.Println("----CURRENT----")
	r.mux.RLock()
	defer r.mux.RUnlock()
	printStats(r.window)
}

func printLogs(a *Accum) {
	//box := goterm.NewBox(20, 3, 0)
	//fmt.Fprint(box, "LOGS")
	//goterm.Println(box.String())
	goterm.Println("----LOGS----")
	a.mux.RLock()
	defer a.mux.RUnlock()
	for _, log := range a.logs {
		goterm.Println(log)
	}
}
