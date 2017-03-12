package command

import (
	"context"
	"sync"

	"flag"

	"strings"

	"github.com/mitchellh/cli"
	"github.com/prokosna/jsongun/internal/lib"
)

type FireCommand struct {
	Ui           cli.Ui
	FilePathList arrayFlags
	Uri          string
	NumShooter   int
	Repeat       int
	Sleep        int
}

type arrayFlags []string

func (af *arrayFlags) String() string {
	return strings.Join(*af, ",")
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

func (c *FireCommand) Run(args []string) int {
	const (
		filePathListDesc  = "Files include lines of JSON. Each JSON should be written in one line."
		uriDefault        = "http://localhost:8081/"
		uriDesc           = "Target URI such as http://localhost:80/"
		numShooterDefault = 1
		numShooterDesc    = "The number of threads to request"
		repeatDefault     = 1
		repeatDesc        = "The number of iteration for one file"
		sleepDefault      = 0
		sleepDesc         = "Interval time between requests"
	)
	cmdFlags := flag.NewFlagSet("fire", flag.ContinueOnError)
	cmdFlags.Usage = func() {
		c.Ui.Output(c.Help())
	}
	cmdFlags.Var(&c.FilePathList, "file", filePathListDesc)
	cmdFlags.Var(&c.FilePathList, "f", filePathListDesc)
	cmdFlags.StringVar(&c.Uri, "uri", uriDefault, uriDesc)
	cmdFlags.StringVar(&c.Uri, "u", uriDefault, uriDesc)
	cmdFlags.IntVar(&c.NumShooter, "num-shooters", numShooterDefault, numShooterDesc)
	cmdFlags.IntVar(&c.NumShooter, "n", numShooterDefault, numShooterDesc)
	cmdFlags.IntVar(&c.Repeat, "repeat", repeatDefault, repeatDesc)
	cmdFlags.IntVar(&c.Repeat, "r", repeatDefault, repeatDesc)
	cmdFlags.IntVar(&c.Sleep, "sleep", sleepDefault, sleepDesc)
	cmdFlags.IntVar(&c.Sleep, "s", sleepDefault, sleepDesc)
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if c.NumShooter <= 0 {
		c.Ui.Error("Please specify the number of shooter not less than 1")
		return 1
	}
	if len(c.FilePathList) <= 0 {
		c.Ui.Error("Please provide at least one file includes JSON with --file option.")
		return 1
	}

	if c.Uri == uriDefault {
		a, err := c.Ui.Ask("Default target 'http://localhost:8081/' will be used. OK? [y/n]")
		if strings.ToLower(a) == "n" || err != nil {
			return 0
		}
	}

	c.Ui.Output("Preparing for shooting...")
	parsers := make([]lib.Parser, len(c.FilePathList))
	for i, filePath := range c.FilePathList {
		parsers[i] = lib.Parser{FilePath: filePath}
	}

	shot, err := lib.NewShot(c.Uri)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// TODO: implement cancel
	ctx, _ := context.WithCancel(context.Background())
	jsonCh := make(chan string, c.NumShooter*3)
	metCh := make(chan lib.Metrics, c.NumShooter*3)
	logCh := make(chan string, c.NumShooter)

	logCh <- "INFO: Start shooting..."

	// file parsers
	var rwg sync.WaitGroup
	for _, parser := range parsers {
		rwg.Add(1)
		go parser.FetchJsonFromFile(ctx, &rwg, jsonCh, logCh, c.Repeat)
	}
	go func() {
		rwg.Wait()
		close(jsonCh)
	}()

	// json shooter
	var wwg sync.WaitGroup
	for i := 0; i < c.NumShooter; i++ {
		wwg.Add(1)
		go shot.Shoot(&wwg, jsonCh, metCh, logCh, c.Sleep)
	}
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
			c.Ui.Output("Done!")
			return 0
		}
	}
}

func (c *FireCommand) Help() string {
	return "Read JSON from the file and POST to target server!"
}

func (c *FireCommand) Synopsis() string {
	return "Fire!"
}
