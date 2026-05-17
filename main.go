package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func main() {
	// Load configuration from .env
	config, err := LoadConfig()
	if err != nil {
		color.Red("❌ Error loading configuration: %v", err)
		os.Exit(1)
	}

	// Initialize OpenRouter client
	client := NewOpenRouterClient(config.APIKey, config.Model)

	// Print welcome banner
	printBanner()

	// Initialize conversation history
	var conversationHistory []Message

	// Main chat loop
	reader := bufio.NewReader(os.Stdin)

	for {
		// Print user input prompt
		color.Cyan("\n👤 You: ")
		fmt.Print("")

		// Read user input
		userInput, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v", err)
			continue
		}

		userInput = strings.TrimSpace(userInput)

		// Handle special commands
		if userInput == "" {
			continue
		}

		if strings.ToLower(userInput) == "/exit" || strings.ToLower(userInput) == "/quit" {
			color.Green("\n✨ Goodbye! Thanks for chatting.\n")
			break
		}

		if strings.ToLower(userInput) == "/clear" {
			conversationHistory = []Message{}
			color.Green("✅ Conversation cleared!\n")
			continue
		}

		if strings.ToLower(userInput) == "/help" {
			printHelp()
			continue
		}

		// Add user message to history
		conversationHistory = append(conversationHistory, Message{
			Role:    "user",
			Content: userInput,
		})

		// Get response from LLM
		color.Yellow("\n🤖 Assistant: ")
		response, err := client.Chat(context.Background(), conversationHistory)
		if err != nil {
			color.Red("Error getting response: %v", err)
			// Remove the last message from history on error
			conversationHistory = conversationHistory[:len(conversationHistory)-1]
			continue
		}

		// Print response
		fmt.Println(response)

		// Add assistant response to history
		conversationHistory = append(conversationHistory, Message{
			Role:    "assistant",
			Content: response,
		})
	}
}

// printBanner prints the welcome banner
func printBanner() {
	color.Cyan("╔════════════════════════════════════════════════════════╗\n")
	color.Cyan("║            🚀 ShellSage - AI Chat CLI v1.0              ║\n")
	color.Cyan("║         Powered by OpenRouter & Cutting-Edge LLMs       ║\n")
	color.Cyan("╚════════════════════════════════════════════════════════╝\n")
	color.White("Type /help for available commands | /exit to quit\n")
}

// printHelp prints available commands
func printHelp() {
	color.Green("\n📚 Available Commands:\n")
	color.White("  /exit  - Exit the chat\n")
	color.White("  /quit  - Exit the chat (alias for /exit)\n")
	color.White("  /clear - Clear conversation history\n")
	color.White("  /help  - Show this help message\n")
}
