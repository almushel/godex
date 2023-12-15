package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/almushel/godex/internal/pokecache"
)

type command struct {
	name, description string
	callback          func(...string) error
}

type pokeEndpoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type LocationAreaList struct {
	Count          int
	Next, Previous string
	Results        []pokeEndpoint
}

type LocationArea struct {
	// First three values don't seem to be in the response?
	//	ID                   int    `json:"id"`
	//	Name                 string `json:"name"`
	//	GameIndex            int    `json:"game_index"`
	EncounterMethodRates []struct {
		EncounterMethod pokeEndpoint `json:"encounter_method"`
		VersionDetails  []struct {
			Rate    int          `json:"rate"`
			Version pokeEndpoint `json:"version"`
		} `json:"version_details"`
	} `json:"encounter_method_rates"`
	Location pokeEndpoint `json:"location"`
	Names    []struct {
		Name     string       `json:"name"`
		Language pokeEndpoint `json:"language"`
	} `json:"names"`
	PokemonEncounters []struct {
		Pokemon        pokeEndpoint `json:"pokemon"`
		VersionDetails []struct {
			Version          pokeEndpoint `json:"version"`
			MaxChance        int          `json:"max_chance"`
			EncounterDetails []struct {
				MinLevel        int          `json:"min_level"`
				MaxLevel        int          `json:"max_level"`
				ConditionValues []any        `json:"condition_values"`
				Chance          int          `json:"chance"`
				Method          pokeEndpoint `json:"method"`
			} `json:"encounter_details"`
		} `json:"version_details"`
	} `json:"pokemon_encounters"`
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
		"explore": {
			name:        "explore",
			description: "Lists the pokemon encounters in a given location area",
			callback:    commandExplore,
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

func commandExit(args ...string) error {
	fmt.Println("Exiting...")
	os.Exit(0)
	return nil
}

func commandHelp(args ...string) error {
	fmt.Print("Usage:\n\n")
	defer fmt.Println("")
	for _, cmd := range appState.commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func getEndpoint[T any](endPointURL, params string) (*T, error) {
	result := new(T)
	getURL := endPointURL + params

	buffer, ok := appState.cache.Get(getURL)
	if !ok {
		response, err := http.Get(getURL)
		if err != nil {
			return result, err
		}
		defer response.Body.Close()

		rb := make([]byte, 1024)

		numRead, err := response.Body.Read(rb)
		for numRead > 0 {
			buffer = append(buffer, rb[:numRead]...)
			if err != nil && err.Error() != "EOF" {
				return result, err
			}
			numRead, err = response.Body.Read(rb)
		}

		appState.cache.Add(getURL, buffer)
	}
	err := json.Unmarshal(buffer, result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func commandMap(args ...string) error {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	locations, err := getEndpoint[LocationAreaList](endPointURL, appState.nextLocations)
	if err != nil {
		return err
	}
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

func commandMapB(args ...string) error {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	if appState.previousLocations == "" {
		return errors.New("No previous location areas to list")
	}
	if appState.previousLocations == endPointURL {
		appState.previousLocations = ""
	}
	locations, err := getEndpoint[LocationAreaList](endPointURL, appState.previousLocations)
	if err != nil {
		return err
	}

	appState.previousLocations = locations.Previous[min(len(endPointURL), len(locations.Previous)):]
	appState.nextLocations = appState.previousLocations

	for _, location := range locations.Results {
		fmt.Println(location.Name)
	}

	return err
}

func commandExplore(args ...string) error {
	const endPointURL = "https://pokeapi.co/api/v2/location-area/"
	if len(args) == 0 || len(args[0]) == 0 {
		return errors.New("Area Unspecified. Usage: explore [area id or name]")
	}
	area, err := getEndpoint[LocationArea](endPointURL, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Exploring %s...\n", args[0])
	fmt.Println("Found Pokemon:")
	for _, pokeman := range area.PokemonEncounters {
		fmt.Printf("  - %s\n", pokeman.Pokemon.Name)
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
		cmdKey, args, _ := strings.Cut(scanner.Text(), " ")
		cmd, ok := appState.commands[cmdKey]
		if !ok {
			fmt.Println("Invalid command:", cmdKey)
			continue
		}
		err := cmd.callback(strings.Split(args, " ")...)
		if err != nil {
			print(err.Error() + "\n")
		}
	}
}
