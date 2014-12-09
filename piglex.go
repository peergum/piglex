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
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	VERSION     = "0.1"
	BLANKSPACES = " \t"
	DEBUG       = false
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
	STATE_ACTIONEND

	USER_STATE
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
	TOKEN_RETURN
	TOKEN_STATE
	TOKEN_TOKEN
	TOKEN_LEN
	TOKEN_VALUE
	TOKEN_ERROR

	USER_TOKEN
)

const (
	CMD_LEX = iota
	CMD_INIT
	CMD_TOKEN
	CMD_STATE
)

var keywords = map[string]int{
	"return": TOKEN_RETURN,
	"state":  TOKEN_STATE,
	"token":  TOKEN_TOKEN,
	"len":    TOKEN_LEN,
	"value":  TOKEN_VALUE,
	"error":  TOKEN_ERROR,
}

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
	tokens   = make([]string, 0, 50)
	states   = []string{"_INIT"}
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
			fmt.Printf("[%d: %s]\n", token.id, token.value)
			result += token.value.(string)
		case <-done:
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
	case STATE_ACTIONEND:
		return lex.stateActionEnd()
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
	if c != '\n' {
		lex.position++
	}
	return
}

func (lex *Lex) checkKeyword() {
	value := lex.getToken().value.(string)
	if len(value) > 0 {
		found := false
		for keyword, tokenId := range keywords {
			if keyword == value {
				token := &Token{
					id:    tokenId,
					char:  0,
					value: value,
				}
				lex.tokens <- token
				logMsg("Token: ", token.value)
				lex.getToken().value = ""
				found = true
				break
			}
		}
		if !found {
			for _, usertoken := range tokens {
				if usertoken == value {
					token := &Token{
						id:    USER_TOKEN,
						char:  0,
						value: value,
					}
					lex.tokens <- token
					logMsg("User token: ", token.value)
					lex.getToken().value = ""
					found = true
					break
				}
			}
		}
		if !found {
			for _, userstate := range states {
				if userstate == value {
					token := &Token{
						id:    USER_STATE,
						char:  0,
						value: value,
					}
					lex.tokens <- token
					logMsg("User state: ", token.value)
					lex.getToken().value = ""
					found = true
					break
				}
			}
		}
		if !found {
			token := &Token{
				id:    TOKEN_ERROR,
				char:  0,
				value: "ERR: " + value,
			}
			lex.tokens <- token
			logMsg("Token: ", token.value)
		}
	}
	token := &Token{
		id:    0,
		char:  0,
		value: "",
	}
	lex.replaceToken(token)
}

func (lex *Lex) checkCommand() {
	value := lex.getToken().value.(string)
	token := &Token{
		id:    TOKEN_INSTRUCTION,
		char:  0,
		value: value,
	}
	lex.tokens <- token
	logMsg("Token: ", token.value)
	fields := strings.Fields(value)
	switch fields[0] {
	case "lex":
		token = &Token{
			id:    0,
			char:  0,
			value: "",
		}
		state := &State{
			current: STATE_LEXRULES,
			token:   token,
		}
		lex.popState()
		lex.replaceState(state)
		return
	case "only":
		logMsg("Only:", strings.Join(fields[1:], " "))
	case "except":
		logMsg("Except:", strings.Join(fields[1:], " "))
	case "include":
		logMsg("Include file:", strings.Join(fields[1:], ", "))
	case "output":
		logMsg("Output file (lexer):", strings.Join(fields[1:], ", "))
	case "token":
		tokenList := strings.Replace(strings.Join(fields[1:], ","), " ", "", -1)
		for _, token := range strings.Split(tokenList, ",") {
			if token != "" {
				tokens = append(tokens, token)
			}
		}
		logMsg("Token(s):", strings.Join(tokens, ", "))
	case "state":
		stateList := strings.Replace(strings.Join(fields[1:], ","), " ", "", -1)
		for _, state := range strings.Split(stateList, ",") {
			if state != "" {
				states = append(states, state)
			}
		}
		logMsg("State(s):", strings.Join(states, ", "))
	case "alias":
		logMsg("Alias:", fields[1], "for", fields[2])
	}
	lex.popState()
}

func (lex *Lex) checkComments(c rune) error {
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
	case c == '#':
		token := &Token{
			id:    '#',
			char:  '#',
			value: string(c),
		}
		state := &State{
			current: STATE_LINECOMMENT,
			token:   token,
		}
		if lex.pushState(state) != nil {
			return errors.New("Oops, can't push state!")
		}
	}

	return nil
}

//
// Basic state
//
func (lex *Lex) stateInit() error {
	logMsg("=== INITIAL STATE ===")
	for lex.getState().current == STATE_INIT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_INIT {
			break
		}
		switch {
		case c == '%' && lex.position == 0:
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_PERCENT,
				token:   token,
			}
			if lex.pushState(state) != nil {
				return errors.New("Oops, can't push state!")
			}
			//lex.tokens <- token
		case c == '\r':
		case c == '\n':
			lex.position = -1
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
			logMsg("Token: ", token.value)
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
		switch {
		case c == '/':
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
		case c == '*':
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
		case strings.IndexRune(BLANKSPACES, c) >= 0:
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
			logMsg("Token: ", token.value)
		}
	}
	return nil
}

//
// C-style comment... expecting */ to leave
//
func (lex *Lex) stateCComment() error {
	logMsg("=== C COMMENT ===")
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
			lex.tokens <- token
			logMsg("Token: ", token.value)
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
			logMsg("Token: ", token.value)

		}
	}
	return nil
}

//
// we're waiting for the end of line
//
func (lex *Lex) stateLineComment() error {
	logMsg("=== INLINE COMMENT ===")
	for lex.getState().current == STATE_LINECOMMENT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		switch c {
		case '\r':
			// do nothing (CR)
		case '\n':
			lex.position = -1
			token := &Token{
				id:    TOKEN_COMMENTLINE,
				char:  0,
				value: lex.getState().token.value,
			}
			lex.popState()
			lex.tokens <- token
			//lex.replaceToken(token)
			logMsg("Token: ", token.value)

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

//
// instruction/command mode
//
func (lex *Lex) statePercent() error {
	logMsg("=== INSTRUCTION STATE ===")
	for lex.getState().current == STATE_PERCENT {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_PERCENT {
			break
		}
		switch {
		case c == '\r':
		case c == '\n':
			lex.position = -1
			lex.checkCommand()
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

//
// regular expression
//
func (lex *Lex) stateLexRules() error {
	logMsg("=== LEX RULES STATE ===")
	for lex.getState().current == STATE_LEXRULES {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_LEXRULES {
			break
		}
		switch {
		case c == '%' && lex.position == 0:
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_PERCENT,
				token:   token,
			}
			lex.pushState(state)
		case (c == '\t' || c == '\n') && lex.position > 0:
			token := &Token{
				id:    TOKEN_REGEXP,
				char:  c,
				value: lex.getToken().value,
			}
			lex.tokens <- token
			logMsg("Token: ", token.value)
			token = &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_ACTION,
				token:   token,
			}
			lex.replaceState(state)
		case c == '\r':
		case c == '\n':
			lex.position = -1
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			lex.replaceToken(token)
			//lex.tokens <- token
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

//
// action
//
func (lex *Lex) stateAction() error {
	logMsg("=== LEX ACTION STATE ===")
	for lex.getState().current == STATE_ACTION {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_ACTION {
			break
		}
		switch {
		case strings.IndexRune(BLANKSPACES, c) >= 0:
			lex.checkKeyword()
		case c == '\r':
		case c == '{' && lex.position > 0:
			token := &Token{
				id:    TOKEN_BLOCKSTART,
				char:  c,
				value: "{",
			}
			lex.tokens <- token
			logMsg("Token: ", token.value)
			token = &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_ACTIONBLOCK,
				token:   token,
			}
			lex.replaceState(state)
		case c == '\n' && lex.getToken().value.(string) != "":
			lex.position = -1
			lex.checkKeyword()
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_LEXRULES,
				token:   token,
			}
			lex.replaceState(state)
		case c == '\n':
			lex.position = -1
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_LEXRULES,
				token:   token,
			}
			lex.replaceState(state)
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

//
// action block
//
func (lex *Lex) stateActionBlock() error {
	logMsg("=== LEX ACTION BLOCK STATE ===")
	for lex.getState().current == STATE_ACTIONBLOCK {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_ACTIONBLOCK {
			break
		}
		switch {
		case strings.IndexRune(BLANKSPACES, c) >= 0 || c == '\n':
			lex.checkKeyword()
			if c == '\n' {
				lex.position = -1
			}
		case c == '\r':
		case c == '}':
			token := &Token{
				id:    TOKEN_BLOCKEND,
				char:  c,
				value: lex.getToken().value.(string) + string("}"),
			}
			lex.tokens <- token
			logMsg("Token: ", token.value)
			token = &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_ACTIONEND,
				token:   token,
			}
			lex.replaceState(state)
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

func (lex *Lex) stateActionEnd() error {
	logMsg("=== LEX ACTION END STATE ===")
	for lex.getState().current == STATE_ACTIONEND {
		c, err := lex.getNext()
		if err != nil {
			return err
		}
		lex.checkComments(c)
		// check if we left init mode
		if lex.getState().current != STATE_ACTIONEND {
			break
		}
		switch {
		case strings.IndexRune(BLANKSPACES, c) >= 0:
		case c == '\r':
		case c == '\n':
			token := &Token{
				id:    0,
				char:  0,
				value: "",
			}
			state := &State{
				current: STATE_LEXRULES,
				token:   token,
			}
			lex.replaceState(state)
		default:
		}
	}
	return nil
}

func (lex *Lex) stateError() error {
	return errors.New(lex.getToken().value.(string))
}

func logMsg(v ...interface{}) {
	if DEBUG {
		log.Println(v)
	}
}
