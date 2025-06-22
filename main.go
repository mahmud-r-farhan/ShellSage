package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Data map[string][]string

func loadData(filepath string) (Data, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var data Data
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func showCategoriesIndexed(data Data) []string {
	fmt.Println("\n📚 Available Categories:")
	keys := []string{}
	i := 1
	for category := range data {
		fmt.Printf(" %d. %s\n", i, category)
		keys = append(keys, category)
		i++
	}
	fmt.Println("\nType the number of a category, or 'search <word>', or 'exit':")
	return keys
}

func showInfo(category string, data Data) {
	items := data[category]
	fmt.Printf("\n🔍 Info for \"%s\":\n", category)
	for _, item := range items {
		fmt.Println("- ", item)
	}
}

func searchQuotes(data Data, keyword string) {
	fmt.Printf("\n🔍 Search Results for \"%s\":\n", keyword)
	found := false
	for category, items := range data {
		for _, item := range items {
			if strings.Contains(strings.ToLower(item), strings.ToLower(keyword)) {
				fmt.Printf("[%s] %s\n", category, item)
				found = true
			}
		}
	}
	if !found {
		fmt.Println("❌ No match found.")
	}
}

func main() {
	data, err := loadData("data/data.json")
	if err != nil {
		fmt.Println("❌ Error loading data:", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		keys := showCategoriesIndexed(data)
		fmt.Print("\n✍️ Enter your choice: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		if strings.HasPrefix(input, "search ") {
			query := strings.TrimPrefix(input, "search ")
			searchQuotes(data, query)
			continue
		}

		index, err := strconv.Atoi(input)
		if err == nil && index >= 1 && index <= len(keys) {
			selected := keys[index-1]
			showInfo(selected, data)

			fmt.Print("\n⏪ Type 'main' to go back or 'exit' to quit: ")
			back, _ := reader.ReadString('\n')
			back = strings.TrimSpace(back)

			if back == "exit" {
				fmt.Println("👋 Goodbye!")
				break
			}
		} else {
			fmt.Println("⚠️ Invalid input. Try again.")
		}
	}
}
