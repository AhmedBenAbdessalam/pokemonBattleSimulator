package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Pokemon struct {
	Id    int
	Name  string
	Stats map[string]int
	Types []Type
	Moves []*Move
}

func (p Pokemon) String() string {
	infos := strings.Builder{}
	infos.WriteString(fmt.Sprintf("Name: %s\n", cases.Title(language.English).String(p.Name)))
	infos.WriteString("Stats:\n")
	for stat, value := range p.Stats {
		infos.WriteString(fmt.Sprintf("  %s: %d\n", stat, value))
	}
	infos.WriteString("Types:\n")
	for _, t := range p.Types {
		infos.WriteString(fmt.Sprintf("  %s\n", t.Name))
		// dont print empty slices
		if len(t.DoubleDamageFrom) > 0 {
			infos.WriteString(fmt.Sprintf("    Double damage from: %v\n", t.DoubleDamageFrom))
		}
		if len(t.DoubleDamageTo) > 0 {
			infos.WriteString(fmt.Sprintf("    Double damage to: %v\n", t.DoubleDamageTo))
		}
		if len(t.HalfDamageFrom) > 0 {
			infos.WriteString(fmt.Sprintf("    Half damage from: %v\n", t.HalfDamageFrom))
		}
		if len(t.HalfDamageTo) > 0 {
			infos.WriteString(fmt.Sprintf("    Half damage to: %v\n", t.HalfDamageTo))
		}
		if len(t.NoDamageFrom) > 0 {
			infos.WriteString(fmt.Sprintf("    No damage from: %v\n", t.NoDamageFrom))
		}
		if len(t.NoDamageTo) > 0 {
			infos.WriteString(fmt.Sprintf("    No damage to: %v\n", t.NoDamageTo))
		}
	}
	infos.WriteString("Moves:\n")
	for _, m := range p.Moves {
		infos.WriteString(fmt.Sprintf("  %s\n", m.Name))
		infos.WriteString(fmt.Sprintf("    Accuracy: %d\n", m.Accuracy))
		infos.WriteString(fmt.Sprintf("    Power: %d\n", m.Power))
		infos.WriteString(fmt.Sprintf("    Pp: %d\n", m.Pp))
		infos.WriteString(fmt.Sprintf("    Type: %s\n", m.Type))
	}
	return infos.String()
}

type Move struct {
	Name     string
	Accuracy int
	Power    int
	Pp       int
	Type     string
}

type Type struct {
	Name             string
	DoubleDamageFrom []string
	DoubleDamageTo   []string
	HalfDamageFrom   []string
	HalfDamageTo     []string
	NoDamageFrom     []string
	NoDamageTo       []string
}

func NewPokemon(pokemonId int) (*Pokemon, error) {
	resp, err := http.Get(fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d", pokemonId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Parse stats
	stats := make(map[string]int)
	for _, stat := range result["stats"].([]interface{}) {
		statObj := stat.(map[string]interface{})
		statName := statObj["stat"].(map[string]interface{})["name"].(string)
		baseStat := int(statObj["base_stat"].(float64))
		stats[statName] = calculateStat(statName, baseStat, 100)
	}

	// Parse types
	types := []Type{}
	for _, t := range result["types"].([]interface{}) {
		typeObj := t.(map[string]interface{})["type"].(map[string]interface{})
		typeUrl := typeObj["url"].(string)
		t, err := fetchType(typeUrl)
		if err != nil {
			return nil, err
		}
		types = append(types, *t)
	}

	// Parse moves
	moves := []*Move{}
	for _, m := range result["moves"].([]interface{}) {
		if len(moves) >= 10 {
			break
		}
		moveObj := m.(map[string]interface{})["move"].(map[string]interface{})
		moveUrl := moveObj["url"].(string)
		move, err := fetchMove(moveUrl)
		if err != nil {
			return nil, err
		}
		if move == nil {
			continue
		}
		moves = append(moves, move)
	}

	// Select 4 random moves
	selectedMoves := selectRandomMoves(moves, 4)

	// Return the populated Pokemon struct
	return &Pokemon{
		Id:    int(result["id"].(float64)),
		Name:  result["name"].(string),
		Stats: stats,
		Types: types,
		Moves: selectedMoves,
	}, nil
}

func calculateStat(statName string, baseStat, level int) int {
	const IV = 31
	const EV = 252
	if statName == "hp" {
		return (((2*baseStat + IV + (EV / 4)) * level) / 100) + level + 10
	}
	return (((2*baseStat + IV + (EV / 4)) * level) / 100) + 5
}

func fetchMove(url string) (*Move, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	moveName := result["name"].(string)
	if result["accuracy"] == nil {
		return nil, nil
	}
	moveAccuracy := int(result["accuracy"].(float64))
	if result["power"] == nil {
		return nil, nil
	}
	movePower := int(result["power"].(float64))
	movePp := int(result["pp"].(float64))
	moveTypeName := result["type"].(map[string]interface{})["name"].(string)

	return &Move{
		Name:     moveName,
		Accuracy: moveAccuracy,
		Power:    movePower,
		Pp:       movePp,
		Type:     moveTypeName,
	}, nil
}

func fetchType(url string) (*Type, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	typeName := result["name"].(string)

	// Parse double damage from
	doubleDamageFrom := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["double_damage_from"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		doubleDamageFrom = append(doubleDamageFrom, typeName)
	}

	// Parse double damage to
	doubleDamageTo := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["double_damage_to"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		doubleDamageTo = append(doubleDamageTo, typeName)
	}

	// Parse half damage from
	halfDamageFrom := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["half_damage_from"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		halfDamageFrom = append(halfDamageFrom, typeName)
	}

	// Parse half damage to
	halfDamageTo := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["half_damage_to"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		halfDamageTo = append(halfDamageTo, typeName)
	}

	// Parse no damage from
	noDamageFrom := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["no_damage_from"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		noDamageFrom = append(noDamageFrom, typeName)
	}

	// Parse no damage to
	noDamageTo := make([]string, 0)
	for _, t := range result["damage_relations"].(map[string]interface{})["no_damage_to"].([]interface{}) {
		typeObj := t.(map[string]interface{})
		typeName := typeObj["name"].(string)
		noDamageTo = append(noDamageTo, typeName)
	}
	return &Type{
		Name:             typeName,
		DoubleDamageFrom: doubleDamageFrom,
		DoubleDamageTo:   doubleDamageTo,
		HalfDamageFrom:   halfDamageFrom,
		HalfDamageTo:     halfDamageTo,
		NoDamageFrom:     noDamageFrom,
		NoDamageTo:       noDamageTo,
	}, nil
}

func selectRandomMoves(possibleMoves []*Move, count int) []*Move {
	if len(possibleMoves) <= count {
		return possibleMoves
	}

	selectedMoves := make([]*Move, count)
	rand.Shuffle(len(possibleMoves), func(i, j int) {
		possibleMoves[i], possibleMoves[j] = possibleMoves[j], possibleMoves[i]
	})

	copy(selectedMoves, possibleMoves[:count])
	return selectedMoves
}

func (p *Pokemon) Attack(enemy *Pokemon) {
	// pick a random move with pp > 0
	var move *Move
	i := 0
	for {
		i++
		move = p.Moves[rand.Intn(len(p.Moves))]
		if move.Pp > 0 {
			break
		}
		if i > 10 {
			fmt.Printf("%s has no moves left\n", p.Name)
			return
		}
	}
	fmt.Printf("%s uses %s\n", p.Name, move.Name)
	move.Pp--

	// Calculate damage using Pokémon damage formula
	damage := calculateDamage(p, *move, enemy)
	if damage < 0 {
		damage = 0
	}

	fmt.Printf("%s takes %d damage\n", enemy.Name, damage)
	enemy.Stats["hp"] -= damage
}

func calculateDamage(attacker *Pokemon, move Move, defender *Pokemon) int {
	level := 100
	power := move.Power
	attack := attacker.Stats["attack"]
	defense := defender.Stats["defense"]

	// Determine if the move is special or physical
	if move.Type == "special" {
		attack = attacker.Stats["special-attack"]
		defense = defender.Stats["special-defense"]
	}

	// Calculate modifier
	modifier := calculateModifier(attacker, move, defender)

	// Calculate base damage
	damage := (((2*level/5)+2)*power*attack/defense)/50 + 2
	return int(float64(damage) * modifier)
}

func calculateModifier(attacker *Pokemon, move Move, defender *Pokemon) float64 {
	modifier := 1.0

	// Apply STAB
	for _, t := range attacker.Types {
		if t.Name == move.Type {
			modifier *= 1.5
			break
		}
	}

	// Apply type effectiveness
	effectiveness := 1.0
	for _, t := range defender.Types {
		for _, t2 := range t.NoDamageFrom {
			if t2 == move.Type {
				effectiveness *= 0
				fmt.Printf("It has no effect on %s\n", defender.Name)
			}
		}
		for _, t2 := range t.HalfDamageFrom {
			if t2 == move.Type {
				effectiveness *= 0.5
				fmt.Printf("It's not very effective on %s\n", defender.Name)
			}
		}
		for _, t2 := range t.DoubleDamageFrom {
			if t2 == move.Type {
				effectiveness *= 2
				fmt.Printf("It's super effective on %s\n", defender.Name)
			}
		}
	}
	modifier *= effectiveness

	// Apply critical hit
	if rand.Float64() < 1.0/24.0 {
		modifier *= 2
		fmt.Println("A critical hit!")
	}

	// Apply random variance
	modifier *= rand.Float64()*0.15 + 0.85

	return modifier
}

func main() {
	for {

		wg := sync.WaitGroup{}
		// Fetch a random Pokémon
		pokemonId := rand.Intn(898) + 1
		pokemon1, err := func() (*Pokemon, error) {
			wg.Add(1)
			defer wg.Done()
			return NewPokemon(pokemonId)
		}()
		if err != nil {
			log.Fatal(err)
		}
		pokemonId = rand.Intn(898) + 1
		pokemon2, err := func() (*Pokemon, error) {
			wg.Add(1)
			defer wg.Done()
			return NewPokemon(pokemonId)
		}()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(pokemon1)
		fmt.Printf("--------------------\n")
		fmt.Println(pokemon2)
		fmt.Printf("--------------------\n\n")
		// let the user place his bets
		fmt.Println("Place your bets!")
		var bet int
		for {
			fmt.Println("1 for", pokemon1.Name)
			fmt.Println("2 for", pokemon2.Name)
			fmt.Scan(&bet)
			if bet == 1 || bet == 2 {
				break
			}
			if bet == 0 {
				return
			}
			fmt.Println("Invalid choice")
		}
		wg.Wait()

		fmt.Printf("%s VS %s\n", pokemon1.Name, pokemon2.Name)
		for pokemon1.Stats["hp"] > 0 && pokemon2.Stats["hp"] > 0 {
			pokemon1.Attack(pokemon2)
			if pokemon2.Stats["hp"] <= 0 {
				break
			}
			pokemon2.Attack(pokemon1)
		}
		fmt.Printf("--------------------\n")
		if pokemon1.Stats["hp"] > 0 {
			fmt.Printf("%s wins!\n", pokemon1.Name)
			if bet == 1 {
				fmt.Println("You win!")
			} else {
				fmt.Println("You lose!")
			}
		} else {
			fmt.Printf("%s wins!\n", pokemon2.Name)
			if bet == 2 {
				fmt.Println("You win!")
			} else {
				fmt.Println("You lose!")
			}
		}
	}
}
