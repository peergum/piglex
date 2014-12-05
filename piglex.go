//
// PigLex
// ------
// Copyright 2014 Philippe Hilger (PeerGum)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	VERSION     = "0.1"
	BLANKSPACES = " \t"
)

const (
	STATE_INIT = iota
	STATE_FINISHED
	STATE_ERROR
	STATE_STACK
	STATE_SLASH
	STATE_STAR
	STATE_LINECOMMENT
	STATE_CCOMMENT
	STATE_PERCENT
	STATE_LEXRULES
	STATE_ACTION
	STATE_ACTIONBLOCK
)

const (
	TOKEN_DOUBLESLASH = 256 + iota
	TOKEN_SLASHSTAR
	TOKEN_STARSLASH
	TOKEN_PERCENT
	TOKEN_COMMENTLINE
	TOKEN_INSTRUCTION
	TOKEN_EOF
	TOKEN_REGEXP
	TOKEN_BLOCKSTART
	TOKEN_ACTIONBLOCK
	TOKEN_ACTION
	TOKEN_BLOCKEND
)

type Token struct {
	id    int
	char  rune
	value interface{}
}

type State struct {
	current int
	token   *Token
}

type Lex struct {
	source   *bufio.Reader
	states   []*State
	tokens   chan *Token
	position int
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

func main() {
	fmt.Printf("Welcome to %s.\n", app)

	if *fName == "" {
		fmt.Println("No file.")
		os.Exit(1)
	}
	var lexFile *os.File
	var err error

	if lexFile, err = os.Open(*fName); err != nil {
		fmt.Printf("Can't open file %s", *fName)
		os.Exit(1)
	}
	defer lexFile.Close()

	lex := Lex{
		source: bufio.NewReader(lexFile),
		states: []*State{
			{
				current: STATE_INIT,
				token: &Token{
					id:    0,
					char:  0,
					value: "",
				},
			},
		},
		tokens: make(chan *Token),
	}

	done := make(chan int)
	go getTokens(lex.tokens, done)

	for !lex.finished() {
		if err := lex.nextState(); err != nil {
			if err != io.EOF {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}
			break
		}
	}
	done <- 0

}

func showVersion() {
	fmt.Printf("%s, version %s", app, VERSION)
	os.Exit(0)
}

func getTokens(tokens chan *Token, done chan int) {
	result := ""
	finished := false
	for !finished {
		select {
		case token := <-tokens:
			fmt.Printf("[%s]", token.value)
			result += token.value.(string)
		case <-done:
			fmt.Println("\nResult:", result)
			finished = true
		}
	}

}

//
// getNewState handles current state
//
func (lex *Lex) nextState() error {
	switch lex.getState().current {
	case STATE_INIT:
		return lex.stateInit()
	case STATE_FINISHED:
	case STATE_STACK:
	case STATE_ERROR:
		return lex.stateError()
	case STATE_SLASH:
		return lex.stateSlash()
	case STATE_CCOMMENT:
		return lex.stateCComment()
	case STATE_STAR:
		return lex.stateStar()
	case STATE_LINECOMMENT:
		return lex.stateLineComment()
	case STATE_PERCENT:
		return lex.statePercent()
	case STATE_LEXRULES:
		return lex.stateLexRules()
	case STATE_ACTION:
		return lex.stateAction()
	case STATE_ACTIONBLOCK:
		return lex.stateActionBlock()
	default:
		return lex.stateError()
	}

	return nil
}

func (lex *Lex) getState() *State {
	currentState := len(lex.states) - 1
	return lex.states[currentState]
}

func (lex *Lex) pushState(state *State) error {
	lex.states = append(lex.states, state)
	//fmt.Printf("\nUp %d: %d\n", len(lex.states), state.current)
	return nil
}

func (lex *Lex) replaceState(state *State) error {
	lex.popState()
	lex.pushState(state)
	return nil
}

func (lex *Lex) popState() {
	currentState := len(lex.states) - 1
	lex.states = lex.states[:currentState]
	//fmt.Printf("\nDown %d: %d\n-- ", len(lex.states), lex.getState().current)
}

func (lex *Lex) getToken() *Token {
	return lex.getState().token
}

func (lex *Lex) replaceToken(token *Token) {
	lex.getState().token = token
}

func (lex *Lex) finished() bool {
	return (lex.getState().current == STATE_FINISHED)
}

func (lex *Lex) setErrorState(error) {
	lex.getState().current = STATE_ERROR
}

func (lex *Lex) printTokenValue() {
	fmt.Println("Token:", lex.getState().token.value)
}

func (lex *Lex) getNext() (c rune, err error) {
	if c, _, err = lex.source.ReadRune(); err != nil {
		lex.tokens <- &Token{
			id:    TOKEN_EOF,
			char:  0,
			value: err.Error(),
		}
		return 0, err
	}
	if c == '\n' {
		lex.position = -1
	} else {
		lex.position++
	}
	return
}

//
// Basic state
//
func (lex *Lex) stateInit() error {
	fmt.Println("\n=== INITIAL STATE ===")
	for lex.getState().current == STATE_INIT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch {
		case c == '/':
			token := &Token{
				id:    '/',
				char:  '/',
				value: c,
			}
			state := &State{
				current: STATE_SLASH,
				token:   token,
			}
			if lex.pushState(state) != nil {
				return errors.New("Oops, can't push state!")
			}
		case c == '%' && lex.position == 0:
			token := &Token{
				id:    '%',
				char:  '%',
				value: string(c),
			}
			state := &State{
				current: STATE_PERCENT,
				token:   token,
			}
			if lex.pushState(state) != nil {
				return errors.New("Oops, can't push state!")
			}
			//lex.tokens <- token
		case strings.IndexRune(BLANKSPACES, c) >= 0:
			// skip blanks
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: string(c),
			}
			lex.replaceToken(token)
			lex.tokens <- token
		}
	}
	return nil
}

//
// slash can be the beginning of a c-style comment or c++ comment line
//
func (lex *Lex) stateSlash() error {
	for lex.getState().current == STATE_SLASH {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch c {
		case '/':
			// line comment
			token := &Token{
				id:    TOKEN_DOUBLESLASH,
				char:  0,
				value: "//",
			}
			state := &State{
				current: STATE_LINECOMMENT,
				token:   token,
			}
			lex.replaceState(state)
			//lex.tokens <- token
		case '*':
			// C style comment
			token := &Token{
				id:    TOKEN_SLASHSTAR,
				char:  0,
				value: "/*",
			}
			state := &State{
				current: STATE_CCOMMENT,
				token:   token,
			}
			lex.replaceState(state)
			//lex.tokens <- token
		default:
			lex.tokens <- lex.getToken()
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.popState()
			lex.replaceToken(token)
			lex.tokens <- token
		}
	}
	return nil
}

//
// C-style comment... expecting */ to leave
//
func (lex *Lex) stateCComment() error {
	fmt.Println("\n=== C COMMENT ===")
	for lex.getState().current == STATE_CCOMMENT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		token := &Token{
			id:    0,
			char:  c,
			value: lex.getToken().value.(string) + string(c),
		}
		switch c {
		case '*':
			state := &State{
				current: STATE_STAR,
				token:   token,
			}
			if lex.pushState(state) != nil {
				return errors.New("Oops, can't push state!")
			}
		default:
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

//
// we're waiting for a slash to leave the comment
//
func (lex *Lex) stateStar() error {
	for lex.getState().current == STATE_STAR {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch c {
		case '/':
			// end of c-style comment
			token := &Token{
				id:    TOKEN_STARSLASH,
				char:  0,
				value: lex.getToken().value.(string) + "/",
			}
			lex.popState()
			lex.popState()
			lex.replaceToken(token)
			lex.tokens <- token
		case '*':
			token := &Token{
				id:    '*',
				char:  '*',
				value: lex.getToken().value.(string) + "*",
			}
			//lex.tokens <- lex.getToken()
			lex.replaceToken(token)
		default:
			lex.tokens <- lex.getToken()
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.popState()
			lex.replaceToken(token)
			lex.tokens <- token
		}
	}
	return nil
}

//
// we're waiting for the end of line
//
func (lex *Lex) stateLineComment() error {
	fmt.Println("\n=== C++ COMMENT ===")
	for lex.getState().current == STATE_LINECOMMENT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch c {
		case '\r':
			// do nothing (CR)
		case '\n':
			token := &Token{
				id:    TOKEN_COMMENTLINE,
				char:  0,
				value: lex.getState().token.value,
			}
			lex.popState()
			lex.replaceToken(token)
			lex.tokens <- token
			//lex.printTokenValue()
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

func (lex *Lex) statePercent() error {
	fmt.Println("\n=== INSTRUCTION STATE ===")

	for lex.getState().current == STATE_PERCENT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch c {
		case '\r':
		case '\n':
			token := &Token{
				id:    TOKEN_INSTRUCTION,
				char:  0,
				value: lex.getToken().value,
			}
			command := strings.Fields(token.value.(string))
			if strings.EqualFold(command[0], "%lex") {
				state := &State{
					current: STATE_LEXRULES,
					token:   token,
				}
				lex.popState()
				lex.replaceState(state)
			} else {
				lex.popState()
				lex.replaceToken(token)
			}
			lex.tokens <- token
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

func (lex *Lex) stateLexRules() error {
	fmt.Println("\n=== LEX RULES STATE ===")
	token := &Token{
		id:    0,
		char:  0,
		value: "",
	}
	lex.replaceToken(token)
	for lex.getState().current == STATE_LEXRULES {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch {
		case c == '%' && lex.position == 0:
			token := &Token{
				id:    '%',
				char:  '%',
				value: string(c),
			}
			state := &State{
				current: STATE_PERCENT,
				token:   token,
			}
			lex.replaceState(state)
		case strings.IndexRune(BLANKSPACES, c) >= 0 || c == '\n':
			token := &Token{
				id:    TOKEN_REGEXP,
				char:  c,
				value: lex.getToken().value.(string),
			}
			state := &State{
				current: STATE_ACTION,
				token:   token,
			}
			lex.replaceState(state)
			lex.tokens <- token
		case c == '\r':
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

func (lex *Lex) stateAction() error {
	fmt.Println("\n=== LEX ACTION STATE ===")
	token := &Token{
		id:    0,
		char:  0,
		value: "",
	}
	lex.replaceToken(token)
	for lex.getState().current == STATE_ACTION {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch {
		case strings.IndexRune(BLANKSPACES, c) >= 0:
		case c == '\r':
		case c == '{':
			token := &Token{
				id:    TOKEN_BLOCKSTART,
				char:  c,
				value: "{",
			}
			state := &State{
				current: STATE_ACTIONBLOCK,
				token:   token,
			}
			lex.replaceState(state)
		case c == '\n':
			lex.position = -1
			token := &Token{
				id:    TOKEN_ACTION,
				char:  c,
				value: lex.getToken().value.(string),
			}
			state := &State{
				current: STATE_LEXRULES,
				token:   token,
			}
			lex.replaceState(state)
			lex.tokens <- token
			fmt.Println("ACTIONEND")
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

func (lex *Lex) stateActionBlock() error {
	fmt.Println("\n=== LEX ACTION BLOCK STATE ===")
	token := &Token{
		id:    0,
		char:  0,
		value: "",
	}
	lex.replaceToken(token)
	for lex.getState().current == STATE_ACTIONBLOCK {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch {
		case strings.IndexRune(BLANKSPACES, c) >= 0:
		case c == '\r':
		case c == '}':
			token := &Token{
				id:    TOKEN_BLOCKEND,
				char:  c,
				value: lex.getToken().value.(string) + string("}"),
			}
			state := &State{
				current: STATE_LEXRULES,
				token:   token,
			}
			lex.replaceState(state)
			lex.tokens <- token
		default:
			token := &Token{
				id:    0,
				char:  c,
				value: lex.getToken().value.(string) + string(c),
			}
			lex.replaceToken(token)
			//lex.tokens <- token
		}
	}
	return nil
}

func (lex *Lex) stateError() error {
	return errors.New(lex.getToken().value.(string))
}
