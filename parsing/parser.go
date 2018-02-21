package parsing

import (
	"bufio"
	"encoding/json"
	"log"
	"strings"
	"sync"
)

const firstCaller string = "null"
const StopSignal = "stop"

func mapLogLineToCall(logLine LogLine) call {
	return call{
		Start:   logLine.Start,
		End:     logLine.End,
		Service: logLine.ServiceName,
		Span:    logLine.Callee,
		Calls:   make([]call, 0),
	}
}

//BuildTree computes the log trace json
func buildTree(logs []LogLine) call {
	callMap := make(map[string][]call)

	for i := len(logs) - 1; i >= 0; i-- {
		calls, ok := callMap[logs[i].Caller]
		newCall := mapLogLineToCall(logs[i])
		if !ok {
			callMap[logs[i].Caller] = []call{newCall}
		} else {
			callMap[logs[i].Caller] = append(calls, newCall)
		}
	}

	transitions := buildTransitions(firstCaller, callMap)

	if len(transitions) > 0 {
		return transitions[0]
	}
	return callMap[logs[0].Caller][0]
}

func buildTransitions(key string, calls map[string][]call) []call {
	result, ok := calls[key]
	if !ok {
		return make([]call, 0)
	}

	for i := range result {
		result[i].Calls = buildTransitions(result[i].Span, calls)
	}

	return result
}

func buildLogResult(id string, logs []LogLine, transformedLogs chan<- string) {
	newLogResult := logResult{Id: id}
	newLogResult.Root = buildTree(logs)
	jsonBytes, err := json.Marshal(newLogResult)

	if err != nil {
		panic(err)
	}
	transformedLogs <- string(jsonBytes)
}

func handleOrphans(orphans map[string][]LogLine, orphanLogs chan<- string) {
	var wg sync.WaitGroup

	for key, logLine := range orphans {
		wg.Add(1)
		go func(serviceName string, logLines []LogLine) {
			defer wg.Done()
			buildLogResult(serviceName, logLines, orphanLogs)
		}(logLine[0].ServiceName, orphans[key])
	}

	wg.Wait()
	orphanLogs <- StopSignal
}

func transformLogs(lines <-chan string, malformedLines, transformedLogs, orphanLogs chan<- string) {
	logs := make(map[string][]LogLine)
	var wg sync.WaitGroup

	for {
		line := <-lines
		if line == StopSignal {
			handleOrphans(logs, orphanLogs)
			break
		}

		parseSucceded, logLine := tryParseLine(line)
		if parseSucceded {
			logID := logLine.Id
			elem, ok := logs[logID]
			if !ok {
				logs[logID] = []LogLine{logLine}
			} else {
				logs[logID] = append(elem, logLine)
			}
			if logLine.Caller == firstCaller {
				//buildLogResult(logLine.ServiceName, logs[logID], transformedLogs)
				wg.Add(1)
				go func(logLines []LogLine) {
					defer wg.Done()
					buildLogResult(logLine.ServiceName, logLines, transformedLogs)
				}(logs[logID])
				delete(logs, logID)
			}
		} else {
			malformedLines <- line
		}
	}

	wg.Wait()
	transformedLogs <- StopSignal
}

func tryParseLine(line string) (bool, LogLine) {
	parts := strings.Split(line, " ")
	if len(parts) != 5 ||
		!strings.Contains(parts[4], "->") {
		return false, *new(LogLine)
	}

	involvedServices := strings.Split(parts[4], "->")
	if len(involvedServices) != 2 {
		return false, *new(LogLine)
	}

	return true, LogLine{
		Start:       strings.Replace(parts[0][:len(parts[0])-1], "T", " ", 1),
		End:         strings.Replace(parts[1][:len(parts[1])-1], "T", " ", 1),
		Id:          parts[2],
		ServiceName: parts[3],
		Caller:      involvedServices[0],
		Callee:      involvedServices[1],
	}
}

func StartWatching(scanner *bufio.Scanner, parsedOutput, orphanOutput, malformedOutput *bufio.Writer) {
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
			text := scanner.Text()

			if text == StopSignal {
				break
			}

			lines <- text
		}

		lines <- StopSignal
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
			lines <- StopSignal
		}
	}()

	go transformLogs(lines, malformedLines, transformedLogs, orphanLogs)

	go func() {
		defer wg.Done()
		for {
			select {
			case transformed := <-transformedLogs:
				if transformed == StopSignal {
					return
				}
				parsedOutput.WriteString(transformed + "\n")
			case orphan := <-orphanLogs:
				if orphan != StopSignal {
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
