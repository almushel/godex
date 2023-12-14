package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/almushel/godex/internal/pokecache"
)

type command struct {
	name, description string
	callback          func() error
}

type LocationAreaList struct {
	Count          int
	Next, Previous string
	Results        []struct {
		Name, URL string
	}
}

var appState struct {
	commands                         map[string]command
	nextLocations, previousLocations string
	cache                            *pokecache.Cache
}

func initCommands() {
	appState.commands = map[string]command{
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
		"map": {
			name:        "map",
			description: "Displays the next 20 locations in the Pokemon world",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the previous 20 locations in the Pokemon world",
			callback:    commandMapB,
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
	for _, cmd := range appState.commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func getLocationAreas(params string) (locations LocationAreaList, err error) {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	getURL := endPointURL + params

	buffer, ok := appState.cache.Get(getURL)
	if !ok {
		response, err := http.Get(getURL)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		defer response.Body.Close()
		defer fmt.Println("")

		rb := make([]byte, 1024)

		numRead, err := response.Body.Read(rb)
		for numRead > 0 {
			buffer = append(buffer, rb[:numRead]...)
			if err != nil && err.Error() != "EOF" {
				return locations, err
			}
			numRead, err = response.Body.Read(rb)
		}

		appState.cache.Add(getURL, buffer)
	}

	locationsPage := new(LocationAreaList)
	err = json.Unmarshal(buffer, locationsPage)
	if err != nil {
		return locations, err
	}
	locations = *locationsPage
	return
}

func commandMap() error {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	locations, err := getLocationAreas(appState.nextLocations)

	if appState.nextLocations != "" {
		appState.previousLocations = appState.nextLocations
	} else {
		appState.previousLocations = endPointURL
	}
	// NOTE: Currently wraps to first page when all location areas have been listed
	appState.nextLocations = locations.Next[min(len(endPointURL), len(locations.Next)):]

	for _, location := range locations.Results {
		fmt.Println(location.Name)
	}

	return err
}

func commandMapB() error {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	if appState.previousLocations == "" {
		return errors.New("No previous location areas to list")
	}
	if appState.previousLocations == endPointURL {
		appState.previousLocations = ""
	}
	locations, err := getLocationAreas(appState.previousLocations)

	appState.previousLocations = locations.Previous[min(len(endPointURL), len(locations.Previous)):]
	appState.nextLocations = appState.previousLocations

	for _, location := range locations.Results {
		fmt.Println(location.Name)
	}

	return err
}

func init() {
	appState.cache = pokecache.NewCache(5 * time.Minute)
	initCommands()
	//appState.nextLocations = "?limit=20&offset=700"
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		cmd, ok := appState.commands[scanner.Text()]
		if !ok {
			fmt.Println("Invalid command:", scanner.Text())
			continue
		}
		err := cmd.callback()
		if err != nil {
			print(err.Error() + "\n")
		}
	}
}
