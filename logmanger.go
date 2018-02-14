package main

import (
	"bufio"
	"log"
	"os"
	"sync"
	"time"

	"github.com/laurentiu-ilici/logmanager/parsing"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func startWatching(scanner *bufio.Scanner, parsedOutput, orphanOutput, malformedOutput *bufio.Writer) {
	bufferSize := 100
	lines := make(chan string, bufferSize)
	malformedLines := make(chan string, bufferSize)
	transformedLogs := make(chan string, bufferSize)
	orphanLogs := make(chan string, bufferSize)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for scanner.Scan() {
			lines <- scanner.Text()
		}

		lines <- parsing.StopSignal

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
			lines <- parsing.StopSignal
		}
	}()

	go parsing.TransformLogs(lines, malformedLines, transformedLogs, orphanLogs)

	go func() {
		defer wg.Done()
		for {
			select {
			case transformed := <-transformedLogs:
				if transformed == parsing.StopSignal {
					return
				}
				parsedOutput.WriteString(transformed + "\n")
			case orphan := <-orphanLogs:
				if orphan != parsing.StopSignal {
					orphanOutput.WriteString(orphan + "\n")
				}
			case malformedLog := <-malformedLines:
				malformedOutput.WriteString(malformedLog + "\n")
			}
		}
	}()

	wg.Wait()
	parsedOutput.Flush()
	orphanOutput.Flush()
	malformedOutput.Flush()
}

func main() {
	start := time.Now()
	pathToRawLogs := "./resources/large-log.txt"
	pathToParsedLogs := "parsed.txt"
	pathToMalformedLogs := "malformed.txt"
	pathToOrphansLogs := "orphans.txt"

	rawLogs, err := os.Open(pathToRawLogs)
	check(err)
	defer rawLogs.Close()

	parsedLogs, err := os.Create(pathToParsedLogs)
	check(err)
	defer parsedLogs.Close()

	orphanLogs, err := os.Create(pathToOrphansLogs)
	check(err)
	defer orphanLogs.Close()

	malformedLogs, err := os.Create(pathToMalformedLogs)
	check(err)
	defer malformedLogs.Close()

	startWatching(bufio.NewScanner(rawLogs), bufio.NewWriter(parsedLogs), bufio.NewWriter(orphanLogs), bufio.NewWriter(malformedLogs))
	elapsed := time.Since(start)
	log.Printf("Parsing took %s", elapsed)
}
