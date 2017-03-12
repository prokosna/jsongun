package lib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Parser struct {
	LinesCount int32
	ErrorCount int32
	FilePath   string
}

func (p Parser) readAndParse(ctx context.Context, q chan string, logCh chan string) bool {
	fp, err := os.Open(p.FilePath)
	if err != nil {
		p.ErrorCount = -1
		logCh <- fmt.Sprintf("ERROR: Cannot open %s", p.FilePath)
		return true
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	c := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return true
		default:
			c += 1
			s := scanner.Text()
			var js interface{}
			if json.Unmarshal([]byte(s), &js) == nil {
				p.LinesCount += 1
				q <- s
			} else {
				logCh <- fmt.Sprintf("WARN: Cannot parse to json. line: %d file: %s body: %s",
					c, p.FilePath, s)
				p.ErrorCount += 1
			}
		}
	}
	if scanner.Err() != nil {
		logCh <- fmt.Sprintf("ERROR: Cannot read to end. %s", p.FilePath)
		p.ErrorCount = -1
	}
	return false
}

func (p Parser) FetchJsonFromFile(ctx context.Context, wg *sync.WaitGroup, q chan string, logCh chan string, r int) {
	defer wg.Done()
	for i := 0; i < r; i++ {
		canceled := p.readAndParse(ctx, q, logCh)
		if canceled {
			return
		}
	}
	return
}
