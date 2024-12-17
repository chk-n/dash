⚠️ NOTE: The language is currently in pre-alpha stage

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
- no GC, no manual memory management 
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
Currently only macos arm64 is supported. 

### MacOS
1. Install llvm version 18
2. Download `dash-<VERSION>-darwin-arm64.pkg` from release and install it
3. Test installation worked by running `dash play`

## Future
- memory management
- import and export libraries
- error handling
- parametric polymorphism
- ad hoc polymorphism 
- built-in build system
- built-in testing framework (fuzzing, property-based testing)
- LSP
- concurrency

## Example

```dash
lib main

import "std/fmt"

fn main() {
    fmt.Println("Hello, World!")
}
```

## Current status
- Only MacOS and Linux supported
- NOT SAFE (yet), no bound checking (requires global error handling)
