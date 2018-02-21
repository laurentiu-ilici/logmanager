package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/laurentiu-ilici/logmanager/parsing"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
		panic("Ups")
	}
}

var config struct {
	RawFile  string
	DestFile string
	Help     bool
}

func init() {
	flag.StringVar(&config.DestFile, "o", "", "output file")
	flag.StringVar(&config.RawFile, "i", "", "input file")
	flag.BoolVar(&config.Help, "h", false, "help")
	flag.Parse()
}

func printHelp() {
	fmt.Println("The program uses either the standard in or a file to read log traces")
	fmt.Println("The default is using the standard in, you can use '-i path/to/your/trace' to give a file on your system")
	fmt.Println("When using the standard in, the process will listen until a 'stop' line is issued")
	fmt.Println("Issuing a stop command terminates the processes and flushes any orphans to the orphans.txt")
	fmt.Println("Any malformed lines in your file will be ignored and output to a malformed.txt in your current directory")
	fmt.Println("Any orphans (calls with no outside root) will be output to an orphans.txt in your current directory")
	fmt.Println("The parsed logs will be output to the standard out by default")
	fmt.Println("You can choose an output file by using the '-o path/to/your/desired/output'")
	fmt.Println("Have fun!")
}

func main() {
	var rawLogs io.Reader
	var parsedLogs io.Writer
	var err error

	if config.Help {
		printHelp()
		return
	}

	pathToMalformedLogs := "malformed.txt"
	pathToOrphansLogs := "orphans.txt"
	orphanLogs, err := os.Create(pathToOrphansLogs)
	check(err)
	defer orphanLogs.Close()

	malformedLogs, err := os.Create(pathToMalformedLogs)
	check(err)
	defer malformedLogs.Close()

	if len(config.RawFile) > 0 {
		fileReader, err := os.Open(config.RawFile)
		check(err)
		rawLogs = fileReader
		defer fileReader.Close()
		fmt.Printf("Processing the input from %q", config.RawFile)
	} else {
		rawLogs = os.Stdin
		fmt.Println("Processing the input from standard input, please run with \"-i pathToFile\" option in order to process a log file")
		fmt.Println("Processing processing will stop once a \"stop\" line is read")
	}

	if len(config.DestFile) > 0 {
		fileWriter, err := os.Create(config.DestFile)
		check(err)
		parsedLogs = fileWriter
		defer fileWriter.Close()
		fmt.Printf("Processing the output to %q", config.DestFile)
	} else {
		parsedLogs = os.Stdout
		fmt.Printf("Processing the output to standard output, please run with -o option in order to output to a file")
	}

	fmt.Println()
	parsing.StartWatching(bufio.NewScanner(rawLogs), bufio.NewWriter(parsedLogs), bufio.NewWriter(orphanLogs), bufio.NewWriter(malformedLogs))
}
