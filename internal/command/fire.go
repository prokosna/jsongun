package command

import (
	"context"
	"sync"

	"github.com/mitchellh/cli"
	"github.com/prokosna/jsongun/internal/lib"
)

type FireCommand struct {
	Ui cli.Ui
}

func (c *FireCommand) Run(_ []string) int {
	c.Ui.Info("Preparing for shooting...")
	p := lib.Parser{FilePath: "./test.json"}
	shot, err := lib.NewShot("http://localhost:8081/")
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// TODO: cancel
	ctx, _ := context.WithCancel(context.Background())
	jsonCh := make(chan string, 16)
	metCh := make(chan lib.Metrics, 16)
	logCh := make(chan string, 16)

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
	go lib.CalcOutputStats(&swg, metCh, logCh)
	done := make(chan bool)
	go func() {
		swg.Wait()
		done <- true
	}()

	for {
		select {
		case <-done:
			return 0
		}
	}
	return 0
}

func (c *FireCommand) Help() string {
	return "Read JSON lines from the file and request POST to target URI"
}

func (c *FireCommand) Synopsis() string {
	return "Fire JSON!"
}
