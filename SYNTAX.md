Syntax
======

##PigLex

**Important**: Part of the syntax for PigYacc is similar to [PigYacc](http://github.com/peergum/pigyacc)'s. Both can be part of the same file without generating
any conflict, and this is even recommended if you use both PigLex and PigYacc.

###comments

* C++ (//...) , C (/* ... */) and bash (#...) style comments are accepted.
* C++ and bash comments are on one line only, but can start anywhere in the line
* C comments can spread on several lines and can start and end anywhere in a line
* C comments cannot be embedded at this time.


###commands

####%include

* **Purpose**: Include source code, useful definitions and macros
* **Usage**: `%include "source"`

####%token

* **Purpose**: Declare tokens that will be generated by piglex, and used in pigyacc
* **Usage**: `%token TOKEN_NAME[, ...]`
* **Note**: Tokens should be in UPPERCASE

####%state

* **Purpose**: Declare states for lexical analysis
* **Usage**: `%state _STATE_NAME[, ...]`
* **Note**: States should be in UPPERCASE and start with an underscore, to differentiate them from tokens

####%output

* **Purpose**: Define the name of the file that will be generated by lex, if any.
* **Usage**: `%output "target"`
* **Note**: The target file is generated by expanding macros defined in the source file.

####%lex

* **Purpose**: Switch the lexical analyser in expression/action mode
* **Usage**: `%lex`

####%only

* **Purpose**: Following rules are valid _exclusively_ in given state(s)
* **Usage**: `%only _STATE_NAME[, ...]`

####%except

* **Purpose**: Following rules are valid in _any state but_ the given one(s)
* **Usage**: `%except _STATE_NAME[, ...]`


###Lexical Rules

####Format

```
regexp	simple_action [simple_action2 ...]

regexp
	simple_action [simple_action2 ...]

regexp	{
	simple_action [simple_action2 ...]
	simple_action3 [...]
	[...]
}
```

####Regular Expressions

Regular expression used are using [RE2 standard](http://code.google.com/p/re2/wiki/Syntax).
The lexer will try to match the longest rule defined.

Use (?i) in front of your expressions if you want them to be case-insensitive, or simply use
-i for a general case-insensitive lexer.

####Recognized Actions

All actions should either change the state of the lexer, return a token or do both. Anything else is also
possible through _macros_.

* `return TOKEN_NAME` returns pre-defined token (%token)
* `state _STATE_NAME` switches the lexer to pre-defined state (%state)
* `macro_name([parameter, ...])` calls macro_`macro_name`

Usable parameters:
* `token` the current token
* `value` the value of the current token (string)
* `'character'` the character value of the token if any

**Notes**:

* The regular expression and the action or action block must be separated by one or more *TAB* character(s)
* Within action blocks, no indentation is necessary, but indenting is recommended anyway, for clarity.
* several actions can fit on the same line, within or without an action block.



_(to be completed)_
