⚠️ NOTE: The language is currently in pre-alpha stage. I am currently bootstrapping the compiler

```
██████╗  █████╗ ███████╗██╗  ██╗
██╔══██╗██╔══██╗██╔════╝██║  ██║
██║  ██║███████║███████╗███████║
██║  ██║██╔══██║╚════██║██╔══██║
██████╔╝██║  ██║███████║██║  ██║
╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
```

A simple, safe and productive programming language for creating fast and reliable software.

## Features

- simple language
- data oriented
- strongly typed and compiled
- memory safe
- immutable data with efficient
	- data construction
 	- data usage (generic structs)
- safe and sound type system
- optional types (no null pointer exceptions)
- limited runtime panics
- no undefined values or behaviour
- no out of bounds reads/writes
- enums
- tagged unions
- rich attribute system
- pattern matching (by type and value)
- higher-order functions
- type validation built into type system
- built-in REPL
- zero cost c interoperability
- compile to
	- arm64
 	- x86_64
- and many more small features :)

## Installation

Run in project root directory to build interpreter:
```
go build cmd/main.go
```

## Example

```dash
lib main

import "std/fmt"

fn main() {
    fmt.println("Hello, World!")
}
```
