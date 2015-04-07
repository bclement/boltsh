package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/bclement/boltsh"
	"github.com/boltdb/bolt"
	"os"
	"strings"
	"time"
)

var raw = flag.Bool("raw", false, "Dump values as byte arrays instead of strings")

type CommandFunc func(level boltsh.Level, args []string) boltsh.Level

/*
ls lists the contents of the provided level
*/
func ls(level boltsh.Level, args []string) boltsh.Level {
	contents := level.List()
	/* TODO formatting */
	for _, entry := range contents {
		fmt.Println(entry)
	}
	return level
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
		/* TODO handle multi-part paths */
		path := args[1]
		var dest boltsh.Level
		if path == ".." {
			dest = level.Prev()
		} else {
			dest = level.Cd(path)
		}
		if dest == nil {
			fmt.Printf("Unable to change directory to %v\n", path)
			rval = level
		} else {
			rval = dest
		}
	}
	return rval
}

/*
get dumps the stored value at the key provided in the arguments.
*/
func get(level boltsh.Level, args []string) boltsh.Level {
	if len(args) < 2 {
		fmt.Printf("Missing key in get command\n")
	} else {
		key := args[1]
		data := level.Get(key)
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
help prints the available commands.
*/
func help(level boltsh.Level, args []string) boltsh.Level {
	/* can't refer to commands map here, causes init loop */
	fmt.Print("\tls - list keys for buckets and values at this level. '/' denotes buckets\n")
	fmt.Print("\tcd - change bucket level. '..' goes back.\n")
	fmt.Print("\tget - dump bucket entry.\n")
	fmt.Print("\texit - exit program.\n")
	return level
}

/*
commands is a mapping of command name to execution function.
*/
var commands = map[string]CommandFunc{
	"ls":   ls,
	"help": help,
	"cd":   cd,
	"get":  get,
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
	err := db.View(eventLoop)
	if err != nil {
		fmt.Printf("Problem viewing database: %v\n", err)
	}
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
		args := strings.Fields(line)
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
