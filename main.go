package main

import (
	"github.com/prokosna/jsongun/internal/jsongun"
	"sync"
	"context"
	"os"
	"fmt"
)

func main() {
	p := jsongun.Parser{FilePath: "./test.json"}
	shot, err := jsongun.NewShot("http://localhost:8081/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	jsonCh := make(chan string)
	metCh := make(chan jsongun.Metrics)
	logCh := make(chan string)

	// file reader
	var rwg sync.WaitGroup
	rwg.Add(1)
	go p.FetchJsonFromFile(ctx, &rwg, jsonCh, logCh)
	go func() {
		rwg.Wait()
		close(jsonCh)
	}()

	// json shooter
	var wwg sync.WaitGroup
	wwg.Add(1)
	go shot.Shoot(&wwg, jsonCh, metCh, logCh)
	go func() {
		wwg.Wait()
		close(metCh)
		close(logCh)
	}()

	// stats calculator
	var swg sync.WaitGroup
	swg.Add(1)
	go jsongun.CalcOutputStats(&swg, metCh, logCh)
	done := make(chan bool)
	go func() {
		swg.Wait()
		done <- true
	}()

	for {
		select {
		case <-done:
			return
		}
	}
}
