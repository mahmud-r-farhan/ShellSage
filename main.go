package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

func main() {
	// Load configuration from .env
	config, err := LoadConfig()
	if err != nil {
		color.Red("❌ Error loading configuration: %v", err)
		os.Exit(1)
	}

	// Print welcome banner
	printBannerEnhanced(config)

	// Initialize conversation state
	state := &ConversationState{
		Messages:    []Message{},
		Persona:     personas["general"],
		Model:       config.Model,
		Temperature: 0.7,
		StartTime:   time.Now(),
	}

	// Initialize OpenRouter client
	client := NewOpenRouterClient(config.APIKey, state.Model)

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

		// Handle empty input
		if userInput == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(strings.ToLower(userInput), "/") {
			cmd := strings.ToLower(userInput)
			parts := strings.Fields(cmd)

			switch parts[0] {
			case "/exit", "/quit":
				printStats(state)
				color.Green("\n✨ Goodbye! Thanks for chatting.\n")
				return

			case "/clear":
				state.Messages = []Message{}
				state.TokenUsage = TokenUsageStats{}
				state.StartTime = time.Now()
				color.Green("✅ Conversation cleared!\n")
				continue

			case "/help":
				printExtendedHelp()
				continue

			case "/model":
				newModel := selectModel(state.Model)
				if newModel != state.Model {
					state.Model = newModel
					client = NewOpenRouterClient(config.APIKey, state.Model)
					color.Green(fmt.Sprintf("✅ Model changed to: %s\n", newModel))
				}
				continue

			case "/persona":
				state.Persona = selectPersona()
				color.Green(fmt.Sprintf("✅ Persona changed to: %s\n", state.Persona.Name))
				continue

			case "/temp":
				state.Temperature = adjustTemperature(state.Temperature)
				color.Green(fmt.Sprintf("✅ Temperature set to: %.2f\n", state.Temperature))
				continue

			case "/save":
				saveConversation(state)
				continue

			case "/load":
				listConversations()
				color.Cyan("\nEnter filename to load: ")
				filename, _ := reader.ReadString('\n')
				filename = strings.TrimSpace(filename)
				if filename != "" {
					if loaded, err := loadConversation(filename); err == nil {
						state = loaded
						state.StartTime = time.Now()
						color.Green("✅ Conversation loaded!\n")
					} else {
						color.Red(fmt.Sprintf("❌ Error loading: %v\n", err))
					}
				}
				continue

			case "/list":
				listConversations()
				continue

			case "/search":
				if len(parts) > 1 {
					query := strings.Join(parts[1:], " ")
					searchConversation(state, query)
				} else {
					color.Yellow("Usage: /search <text>\n")
				}
				continue

			case "/stats":
				printStats(state)
				continue

			case "/copy":
				if len(state.Messages) > 0 && state.Messages[len(state.Messages)-1].Role == "assistant" {
					lastMessage := state.Messages[len(state.Messages)-1].Content
					// Copy to clipboard (basic implementation)
					color.Green(fmt.Sprintf("✅ Last response:\n%s\n", lastMessage))
				} else {
					color.Yellow("No assistant message to copy.\n")
				}
				continue

			case "/history":
				showHistory(state, 10)
				continue

			default:
				color.Yellow(fmt.Sprintf("❌ Unknown command: %s\n", parts[0]))
				color.White("Type /help for available commands.\n")
				continue
			}
		}

		// Add user message to history
		state.Messages = append(state.Messages, Message{
			Role:    "user",
			Content: userInput,
		})

		// Build messages with system prompt
		messagesWithSystem := []Message{
			{
				Role:    "system",
				Content: state.Persona.Prompt,
			},
		}
		messagesWithSystem = append(messagesWithSystem, state.Messages...)

		// Get response from LLM with usage tracking
		color.Yellow("\n🤖 Assistant: ")
		response, usage, err := client.ChatWithUsage(context.Background(), messagesWithSystem, state.Temperature)
		if err != nil {
			color.Red(fmt.Sprintf("❌ Error getting response: %v\n", err))
			// Remove the last message from history on error
			state.Messages = state.Messages[:len(state.Messages)-1]
			continue
		}

		// Print response
		fmt.Println(response)

		// Add assistant response to history
		state.Messages = append(state.Messages, Message{
			Role:    "assistant",
			Content: response,
		})

		// Update token usage
		if usage != nil {
			state.TokenUsage.TotalPromptTokens += usage.PromptTokens
			state.TokenUsage.TotalCompletionTokens += usage.CompletionTokens
			state.TokenUsage.TotalTokens += usage.TotalTokens
			state.TokenUsage.MessagesCount = len(state.Messages) / 2

			color.Magenta(fmt.Sprintf("\n[Tokens: +%d prompt, +%d completion | Total: %d]\n",
				usage.PromptTokens, usage.CompletionTokens, state.TokenUsage.TotalTokens))
		}
	}
}
