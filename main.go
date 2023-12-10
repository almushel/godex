package main

import (
	"bufio"
	"fmt"
	"os"
)

type command struct {
	name, description string
	callback          func() error
}

var commands map[string]command

func initCommands() {
	commands = map[string]command{
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"exit": {
			name:        "exit",
			description: "Exits the program",
			callback:    commandExit,
		},
	}
}

func commandExit() error {
	fmt.Println("Exiting...")
	os.Exit(0)
	return nil
}

func commandHelp() error {
	fmt.Print("Usage:\n\n")
	defer fmt.Println("")
	for _, cmd := range commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func main() {
	initCommands()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		cmd, ok := commands[scanner.Text()]
		if !ok {
			fmt.Println("Invalid command:", scanner.Text())
			continue
		}
		cmd.callback()
	}
}
