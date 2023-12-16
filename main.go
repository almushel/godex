package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

var appState struct {
	commands                         map[string]command
	nextLocations, previousLocations string
	cache                            *pokecache.Cache

	pokemon map[string]Pokemon
}

func initCommands() {
	appState.commands = map[string]command{
		"catch": {
			name:        "catch",
			description: "Attempt to catch named Pokemon",
			callback:    commandCatch,
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
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"inspect": {
			name:        "inspect",
			description: "Display information about a previously caught Pokemon",
			callback:    commandInspect,
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

func commandCatch(args ...string) error {
	const endPointURL = "https://pokeapi.co/api/v2/pokemon/"
	if len(args) == 0 || len(args[0]) == 0 {
		return errors.New("Pokemon unspecified. Usage: catch [pokemon id or name]")
	}

	pokemon, err := getEndpoint[Pokemon](endPointURL, args[0])
	if err != nil {
		return err
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", args[0])

	var catchChance float32 = 100 / float32(pokemon.BaseExperience*pokemon.BaseExperience)
	catchChance *= 2000
	var throw float32 = rand.Float32() * 100
	//fmt.Println("catch chance:", catchChance+1, ", throw:", throw)
	if throw <= catchChance+1 {
		fmt.Println(args[0], "was caught!")
		appState.pokemon[args[0]] = *pokemon
	} else {
		fmt.Println(args[0], "escaped!")
	}

	return nil
}

func commandExit(args ...string) error {
	fmt.Println("Exiting...")
	os.Exit(0)
	return nil
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

func commandHelp(args ...string) error {
	fmt.Print("Usage:\n\n")
	defer fmt.Println("")
	for _, cmd := range appState.commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandInspect(args ...string) error {
	if len(args) == 0 || len(args[0]) == 0 {
		return errors.New("No pokemon specified. Usage: inspect [pokemon name or id]")
	}
	p, ok := appState.pokemon[args[0]]
	if !ok {
		fmt.Println("You have not caught that Pokemon")
		return nil
	}
	fmt.Println(args[0])
	fmt.Println("Height:", p.Height)
	fmt.Println("Weight:", p.Weight)
	fmt.Println("Stats:")
	for _, stat := range p.Stats {
		fmt.Printf("   - %s: %d\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, pType := range p.Types {
		fmt.Printf("   - %s\n", pType.Type.Name)
	}

	return nil
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

func init() {
	appState.cache = pokecache.NewCache(5 * time.Minute)
	appState.pokemon = make(map[string]Pokemon)
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
