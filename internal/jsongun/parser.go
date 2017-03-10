package jsongun

import (
	"context"
	"sync"
	"os"
	"bufio"
	"encoding/json"
	"fmt"
)

type Parser struct {
	LinesCount int32
	ErrorCount int32
	FilePath   string
}

func (p *Parser) FetchJsonFromFile(ctx context.Context, wg *sync.WaitGroup, q chan string, logCh chan string) {
	defer wg.Done()
	fp, err := os.Open(p.FilePath)
	if err != nil {
		p.ErrorCount = -1
		logCh <- fmt.Sprintf("ERROR: Cannot open %s", p.FilePath)
		return;
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	c := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			c += 1;
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
	return
}
