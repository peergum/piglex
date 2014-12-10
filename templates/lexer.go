package main

import (
	"bufio"
	//"errors"
	"flag"
	"fmt"
	"io"
	//"log"
	"os"
	"path/filepath"
	"regexp"
	//"strings"
)

const (
	VERSION = "0.1"
	DEBUG   = false
)

type Rule struct {
	regexp string
	action func(string) error
}

var (
	app      string  = filepath.Base(os.Args[0])
	fVersion *bool   = flag.Bool("v", false, "Show Version")
	fName    *string = flag.String("f", "", "File to parse")
	flags    []string
	args     []string
)

func init() {
	flag.Parse()
	args = flag.Args()

	if *fVersion {
		showVersion()
	}
}

func showVersion() {
	fmt.Printf("%s, version %s", app, VERSION)
	os.Exit(0)
}

func main() {
	fmt.Printf("Welcome to %s.\n", app)

	if *fName == "" {
		fmt.Println("No file.")
		os.Exit(1)
	}
	var source *os.File
	var err error

	if source, err = os.Open(*fName); err != nil {
		fmt.Printf("Can't open file %s", *fName)
		os.Exit(1)
	}
	defer source.Close()

	loop(bufio.NewReader(source))

}

func loop(reader *bufio.Reader) error {
	position := 0
	bestMatch := 0
	current := ""
	var bestRule *Rule
	for state != "_END" {
		if position > 0 {
			current = current[position:]
			position = 0
			bestRule = nil
		} else {
			next, _, err := reader.ReadRune()
			if err != nil {
				if err != io.EOF {
					panic("Read Error @ [" + current + "]")
				}
				if bestRule != nil {
					bestRule.call(current)
					break
				}
			}
			current += string(next)
		}
		for _, rule := range rules[state] {
			regexp := regexp.MustCompilePOSIX("^" + rule.regexp + "$")
			match := regexp.FindString(current)
			if len(match) > bestMatch {
				bestMatch = len(match)
				bestRule = rule
			}
		}
		if bestRule == nil {
			if bestMatch == 0 {
				panic("SYNTAX ERROR @ [" + current + "]")
			}
			position = bestMatch
			bestRule.call(current[0:position])
		}
	}
	return nil
}

func (rule *Rule) call(value string) error {
	return rule.action(value)
}
