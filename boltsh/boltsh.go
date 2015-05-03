package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bclement/boltsh"
	"github.com/boltdb/bolt"
)

var raw = flag.Bool("raw", false, "Dump values as byte arrays instead of strings")

type CommandFunc func(level boltsh.Level, args []string) boltsh.Level

/*
ls lists the contents of the provided level
*/
func ls(level boltsh.Level, args []string) boltsh.Level {
	rval := level
	if len(args) > 1 {
		level = travel(level, args[1])
	}

	if level != nil {
		contents := level.List()
		/* TODO formatting */
		for _, entry := range contents {
			fmt.Println(entry)
		}
	} else {
		fmt.Printf("Unable to list path %v\n", args[1])
	}
	return rval
}

/*
cd returns the level according to the value provided in args.
If the value is not valid, the same level is returned.
If no values are given in the arguments, the root level is returned.
*/
func cd(level boltsh.Level, args []string) boltsh.Level {
	var rval boltsh.Level
	if len(args) < 2 {
		/* go to root */
		parent := level.Prev()
		for parent != nil {
			rval = parent
			parent = rval.Prev()
		}
	} else {
		path := args[1]
		curr := travel(level, path)
		if curr == nil {
			fmt.Printf("Unable to change directory to %v\n", path)
			rval = level
		} else {
			rval = curr
		}
	}
	return rval
}

/* travel splits the path by slashes and walks down to the specified level.
returns nil if any part of the path is invalid
*/
func travel(level boltsh.Level, path string) boltsh.Level {
	parts := strings.Split(path, "/")
	for i := 0; level != nil && i < len(parts); i += 1 {
		part := parts[i]
		if part == ".." {
			level = level.Prev()
		} else if part != "." {
			level = level.Cd(part)
		}
	}
	return level
}

/*
get dumps the stored value at the key provided in the arguments.
*/
func get(level boltsh.Level, args []string) boltsh.Level {
	if len(args) < 2 {
		fmt.Printf("Missing key in get command\n")
	} else {
		target, key := parseKeyPath(level, args[1])
		var data []byte
		if target != nil {
			data = target.Get(key)
		}
		if data == nil {
			fmt.Printf("No data at key %v\n", key)
		} else {
			if *raw {
				fmt.Printf("%v\n", data)
			} else {
				fmt.Printf("%v\n", string(data))
			}
		}
	}
	return level
}

/*
parseKeyPath walks the path until the last path part.
The new level and key (last path part) are returned.
*/
func parseKeyPath(level boltsh.Level, path string) (boltsh.Level, string) {
	slashIndex := strings.LastIndex(path, "/")
	var key string
	if slashIndex < 0 {
		key = path
	} else {
		key = path[slashIndex+1:]
		level = travel(level, path[:slashIndex])
	}
	return level, key
}

/*
put expects two args, a path key and a value.
The value is stored at the key and the provided level is returned.
*/
func put(level boltsh.Level, args []string) boltsh.Level {
	if len(args) < 3 {
		fmt.Printf("Put command must specify key and value\n")
	} else {
		target, key := parseKeyPath(level, args[1])
		if target == nil {
			fmt.Printf("Unable to put %v at path %v\n", args[2], args[1])
		} else {
			target.Put(key, args[2])
		}
	}
	return level
}

/*
mkdir expects one arg, a path key.
A new bucket is created at the key.
*/
func mkdir(level boltsh.Level, args []string) boltsh.Level {
	if len(args) < 2 {
		fmt.Printf("Mkdir command must specify key\n")
	} else {
		target, key := parseKeyPath(level, args[1])
		if target == nil {
			fmt.Printf("Unable to create bucket at path %v\n", args[1])
		} else {
			target.Mkdir(key)
		}
	}
	return level
}

/*
help prints the available commands.
*/
func help(level boltsh.Level, args []string) boltsh.Level {
	/* can't refer to commands map here, causes init loop */
	fmt.Print("\tls [path] - list keys for buckets and values at this level. '/' denotes buckets\n")
	fmt.Print("\tcd [path] - change bucket level. '..' goes back.\n")
	fmt.Print("\tget [path] - dump bucket entry.\n")
	fmt.Print("\tmkdir [path] - creates a new bucket.\n")
	fmt.Print("\tput [path] [json] - add bucket entry.\n")
	fmt.Print("\texit - exit program.\n")
	return level
}

/*
commands is a mapping of command name to execution function.
*/
var commands = map[string]CommandFunc{
	"ls":    ls,
	"help":  help,
	"cd":    cd,
	"get":   get,
	"put":   put,
	"mkdir": mkdir,
}

func main() {

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Printf("Usage: %v [db file]\n", os.Args[0])
		os.Exit(1)
	}

	fname := args[0]
	db := open(fname)
	defer db.Close()

	fmt.Println("Type 'help' for list of commands")
	err := db.Update(eventLoop)
	if err != nil {
		fmt.Printf("Problem viewing database: %v\n", err)
	}
}

const (
	OUTSIDE = iota
	INQUOTE
	INWORD
)

func argSplit(line string) []string {
	state := OUTSIDE
	var rval []string
	var buff bytes.Buffer
	var ch rune
	for i, w := 0, 0; i < len(line); i += w {
		ch, w = utf8.DecodeRuneInString(line[i:])
		if state == OUTSIDE {
			if ch == '"' {
				state = INQUOTE
			} else if !unicode.IsSpace(ch) {
				state = INWORD
				buff.WriteRune(ch)
			}
		} else if state == INQUOTE {
			if ch == '\\' {
				next, width := utf8.DecodeRuneInString(line[i+w:])
				if next == '"' {
					i, w = i+w, width
					buff.WriteRune('"')
				}
			} else if ch == '"' {
				state = OUTSIDE
				rval = append(rval, buff.String())
				buff.Reset()
			} else {
				buff.WriteRune(ch)
			}
		} else if state == INWORD {
			if ch == '"' {
				state = INQUOTE
			} else if unicode.IsSpace(ch) {
				state = OUTSIDE
				rval = append(rval, buff.String())
				buff.Reset()
			} else {
				buff.WriteRune(ch)
			}
		}
	}
	last := buff.String()
	if last != "" {
		rval = append(rval, last)
	}
	return rval
}

/*
eventLoop runs the main logic of the program in a DB transaction.
*/
func eventLoop(tx *bolt.Tx) error {
	reader := bufio.NewReader(os.Stdin)
	var level boltsh.Level
	level = boltsh.NewRootLevel(tx)
	for {
		fmt.Print("$ ")
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil || line == "exit" {
			/* TODO output error if not EOF */
			fmt.Println()
			break
		}
		args := argSplit(line)
		if len(args) > 0 {
			cmd, ok := commands[args[0]]
			if !ok {
				fmt.Printf("Unrecognized command: %v\n", args[0])
			} else {
				level = cmd(level, args)
			}
		}
	}
	return nil
}

/*
open returns a database object or doesn't return at all.
*/
func open(fname string) *bolt.DB {
	if _, err := os.Stat(fname); err != nil {
		fmt.Printf("Unable to stat database file %v\n%v\n", fname, err)
		os.Exit(1)
	}
	db, err := bolt.Open(fname, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		fmt.Printf("Unable to open database file: %v\n%v\n", fname, err)
		os.Exit(1)
	}
	return db
}
