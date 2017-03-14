package command

import (
	"context"
	"sync"

	"flag"

	"strings"

	"fmt"

	"github.com/mitchellh/cli"
	"github.com/prokosna/jsongun/internal/lib"
)

type FireCommand struct {
	Ui cli.Ui
}

type Config struct {
	filePathList arrayFlags
	uri          string
	numShooter   uint
	repeat       uint
	sleep        uint
	flushRate    uint
}

type arrayFlags []string

func (af *arrayFlags) String() string {
	return strings.Join(*af, ",")
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

var cmdFlags *flag.FlagSet
var cfg *Config

const (
	uriDefault        = "http://localhost:8081/"
	uriDesc           = "Target URI"
	numShooterDefault = 1
	numShooterDesc    = "The number of threads to request (> 0)"
	repeatDefault     = 1
	repeatDesc        = "The number of iteration for each file (> 0)"
	sleepDefault      = 0
	sleepDesc         = "Interval between requests (milliseconds, > 0)"
	flushRateDefault  = 1000
	flushRateDesc     = "Flush stats rate (milliseconds, > 0)"
)

func init() {
	cfg = &Config{}
	cmdFlags = flag.NewFlagSet("fire", flag.ContinueOnError)
	cmdFlags.StringVar(&cfg.uri, "uri", uriDefault, uriDesc)
	cmdFlags.StringVar(&cfg.uri, "u", uriDefault, uriDesc)
	cmdFlags.UintVar(&cfg.numShooter, "num-shooters", numShooterDefault, numShooterDesc)
	cmdFlags.UintVar(&cfg.numShooter, "n", numShooterDefault, numShooterDesc)
	cmdFlags.UintVar(&cfg.repeat, "repeat", repeatDefault, repeatDesc)
	cmdFlags.UintVar(&cfg.repeat, "r", repeatDefault, repeatDesc)
	cmdFlags.UintVar(&cfg.sleep, "sleep", sleepDefault, sleepDesc)
	cmdFlags.UintVar(&cfg.sleep, "s", sleepDefault, sleepDesc)
	cmdFlags.UintVar(&cfg.flushRate, "flushRate", flushRateDefault, flushRateDesc)
	cmdFlags.UintVar(&cfg.flushRate, "f", flushRateDefault, flushRateDesc)
}

func (c *FireCommand) Run(args []string) int {
	cmdFlags.Usage = func() {
		c.Ui.Output(c.Help())
	}
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	cfg.filePathList = cmdFlags.Args()
	if cfg.numShooter <= 0 {
		c.Ui.Error("Please enter the number of shooter greater than 1")
		c.Ui.Output(c.Help())
		return 1
	}
	if len(cfg.filePathList) <= 0 {
		c.Ui.Error("Please enter the JSON files")
		c.Ui.Output(c.Help())
		return 1
	}
	if cfg.uri == uriDefault {
		a, err := c.Ui.Ask("Default target 'http://localhost:8081/' will be used. OK? [y/n]")
		if strings.ToLower(a) == "n" || err != nil {
			return 0
		}
	}
	if cfg.flushRate <= 0 {
		c.Ui.Error("Please enter flush rate greater than 1")
		c.Ui.Output(c.Help())
		return 1
	}

	c.Ui.Output("Preparing for shooting...")
	parsers := make([]lib.Parser, len(cfg.filePathList))
	for i, filePath := range cfg.filePathList {
		parsers[i] = lib.Parser{FilePath: filePath}
	}

	shot, err := lib.NewShot(cfg.uri)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// TODO: implement cancel
	ctx, _ := context.WithCancel(context.Background())
	jsonCh := make(chan string, cfg.numShooter*3)
	metCh := make(chan lib.Metrics, cfg.numShooter*3)
	logCh := make(chan string, cfg.numShooter)

	logCh <- "INFO: Start shooting..."

	// file parsers
	var rwg sync.WaitGroup
	for _, parser := range parsers {
		rwg.Add(1)
		go parser.FetchJsonFromFile(ctx, &rwg, jsonCh, logCh, int(cfg.repeat))
	}
	go func() {
		rwg.Wait()
		close(jsonCh)
	}()

	// json shooter
	var wwg sync.WaitGroup
	for i := 0; i < int(cfg.numShooter); i++ {
		wwg.Add(1)
		go shot.Shoot(&wwg, jsonCh, metCh, logCh, int(cfg.sleep))
	}
	go func() {
		wwg.Wait()
		close(metCh)
		close(logCh)
	}()

	// stats calculator
	var swg sync.WaitGroup
	swg.Add(1)
	go lib.CalcOutputStats(&swg, int(cfg.flushRate), metCh, logCh)
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
	text := fmt.Sprintln("Usage: jsongun fire [options] FILES...")
	opt := fmt.Sprintln("Options:")
	args := make([]string, 6)
	args[1] = fmt.Sprintf("  %s, %s\n\t%s [%s]\n", "-u", "--uri", uriDesc, uriDefault)
	args[2] = fmt.Sprintf("  %s, %s\n\t%s [%d]\n", "-n", "--num-shooters", numShooterDesc, numShooterDefault)
	args[3] = fmt.Sprintf("  %s, %s\n\t%s [%d]\n", "-r", "--repeat", repeatDesc, repeatDefault)
	args[4] = fmt.Sprintf("  %s, %s\n\t%s [%d]\n", "-s", "--sleep", sleepDesc, sleepDefault)
	args[5] = fmt.Sprintf("  %s, %s\n\t%s [%d]\n", "-f", "--flushRate", flushRateDesc, flushRateDefault)
	return text + opt + strings.Join(args, "")
}

func (c *FireCommand) Synopsis() string {
	return "Read JSON from the file and POST to the target!"
}
