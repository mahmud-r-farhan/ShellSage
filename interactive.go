package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Persona represents an AI persona with system prompt
type Persona struct {
	Name   string
	Prompt string
}

// Prebuilt personas
var personas = map[string]Persona{
	"general": {
		Name:   "General Assistant",
		Prompt: "You are a helpful, respectful, and honest assistant. Provide clear and concise answers.",
	},
	"developer": {
		Name:   "Expert Developer",
		Prompt: "You are an expert software engineer. Provide detailed code examples, best practices, and performance insights.",
	},
	"writer": {
		Name:   "Creative Writer",
		Prompt: "You are a creative writing expert. Help with storytelling, character development, and narrative structure.",
	},
	"teacher": {
		Name:   "Patient Teacher",
		Prompt: "You are a patient and encouraging teacher. Explain concepts clearly with examples, adapt to the learner's level.",
	},
	"analyst": {
		Name:   "Data Analyst",
		Prompt: "You are a skilled data analyst. Help analyze trends, create visualizations, and draw insights from data.",
	},
	"debug": {
		Name:   "Debug Assistant",
		Prompt: "You are an expert at debugging. Help identify and fix issues in code, provide step-by-step troubleshooting.",
	},
}

// ConversationState holds the current conversation
type ConversationState struct {
	Messages    []Message
	Persona     Persona
	Model       string
	Temperature float64
	SavePath    string
	TokenUsage  TokenUsageStats
	StartTime   time.Time
}

// TokenUsageStats tracks token usage
type TokenUsageStats struct {
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
	MessagesCount         int
}

// selectModel shows an interactive model selector
func selectModel(currentModel string) string {
	reader := bufio.NewReader(os.Stdin)
	models := []string{
		"meta-llama/llama-2-70b-chat",
		"mistralai/mistral-7b-instruct",
		"openrouter/free",
		"gpt-3.5-turbo",
		"claude-3-haiku",
	}

	color.Cyan("\n📊 Available Models:\n")
	for i, model := range models {
		color.White(fmt.Sprintf("  [%d] %s\n", i+1, model))
	}
	color.White(fmt.Sprintf("  [0] Keep current: %s\n", currentModel))

	color.Cyan("\nSelect model (0 to skip): ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "0" || choice == "" {
		return currentModel
	}

	if len(choice) > 0 && choice[0] >= '1' && choice[0] <= byte(len(models)+'0') {
		return models[choice[0]-'1']
	}

	return choice
}

// selectPersona shows an interactive persona selector
func selectPersona() Persona {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan("\n🎭 Available Personas:\n")
	keys := make([]string, 0, len(personas))
	for key := range personas {
		keys = append(keys, key)
	}

	for i, key := range keys {
		color.White(fmt.Sprintf("  [%d] %s\n", i+1, personas[key].Name))
	}

	color.Cyan("\nSelect persona (default=1): ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "" {
		choice = "1"
	}

	if len(choice) > 0 && choice[0] >= '1' && choice[0] <= byte(len(personas)+'0') {
		return personas[keys[choice[0]-'1']]
	}

	return personas["general"]
}

// adjustTemperature allows user to adjust temperature
func adjustTemperature(current float64) float64 {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan(fmt.Sprintf("\n🌡️  Current temperature: %.2f (0.0 = focused, 1.0 = creative)\n", current))
	color.Cyan("Enter new value (0.0-1.0) or press Enter to skip: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current
	}

	var temp float64
	_, err := fmt.Sscanf(input, "%f", &temp)
	if err != nil || temp < 0.0 || temp > 1.0 {
		color.Red("Invalid value! Keeping current temperature.\n")
		return current
	}

	return temp
}

// saveConversation saves conversation to file
func saveConversation(state *ConversationState) {
	if len(state.Messages) == 0 {
		color.Yellow("⚠️  No messages to save.\n")
		return
	}

	filename := fmt.Sprintf("conversations/chat_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	_ = os.MkdirAll("conversations", 0755)

	data := map[string]interface{}{
		"timestamp": time.Now(),
		"model":     state.Model,
		"persona":   state.Persona.Name,
		"messages":  state.Messages,
		"stats":     state.TokenUsage,
	}

	bytes, _ := json.MarshalIndent(data, "", "  ")
	_ = os.WriteFile(filename, bytes, 0644)

	color.Green(fmt.Sprintf("✅ Conversation saved to %s\n", filename))
}

// listConversations lists saved conversations
func listConversations() {
	entries, err := os.ReadDir("conversations")
	if err != nil {
		color.Yellow("No saved conversations found.\n")
		return
	}

	color.Cyan("\n💾 Saved Conversations:\n")
	for i, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			color.White(fmt.Sprintf("  [%d] %s (%d bytes)\n", i+1, entry.Name(), info.Size()))
		}
	}
}

// loadConversation loads a conversation from file
func loadConversation(filename string) (*ConversationState, error) {
	path := filepath.Join("conversations", filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	state := &ConversationState{}
	if messagesData, ok := data["messages"].([]interface{}); ok {
		for _, msg := range messagesData {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				state.Messages = append(state.Messages, Message{
					Role:    msgMap["role"].(string),
					Content: msgMap["content"].(string),
				})
			}
		}
	}

	return state, nil
}

// searchConversation searches through conversation history
func searchConversation(state *ConversationState, query string) {
	color.Cyan("\n🔍 Search Results:\n")
	found := 0
	query = strings.ToLower(query)

	for i, msg := range state.Messages {
		if strings.Contains(strings.ToLower(msg.Content), query) {
			found++
			color.Yellow(fmt.Sprintf("\n[Message %d] %s:\n", i+1, msg.Role))
			color.White(msg.Content + "\n")
		}
	}

	if found == 0 {
		color.Yellow("No messages matching your search.\n")
	} else {
		color.Green(fmt.Sprintf("\n✅ Found %d matching message(s)\n", found))
	}
}

// printStats displays token usage and conversation stats
func printStats(state *ConversationState) {
	elapsed := time.Since(state.StartTime)
	color.Cyan("\n📊 Conversation Statistics:\n")
	color.White(fmt.Sprintf("  Duration: %v\n", elapsed))
	color.White(fmt.Sprintf("  Messages: %d\n", len(state.Messages)))
	color.White(fmt.Sprintf("  Persona: %s\n", state.Persona.Name))
	color.White(fmt.Sprintf("  Model: %s\n", state.Model))
	color.White(fmt.Sprintf("  Temperature: %.2f\n", state.Temperature))
	color.Cyan("\n📈 Token Usage:\n")
	color.White(fmt.Sprintf("  Prompt Tokens: %d\n", state.TokenUsage.TotalPromptTokens))
	color.White(fmt.Sprintf("  Completion Tokens: %d\n", state.TokenUsage.TotalCompletionTokens))
	color.White(fmt.Sprintf("  Total Tokens: %d\n", state.TokenUsage.TotalTokens))
}

// printExtendedHelp prints detailed help
func printExtendedHelp() {
	color.Green("\n📚 Enhanced Commands:\n\n")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/help", "Show this help message"},
		{"/exit", "Exit the chat"},
		{"/quit", "Exit the chat (alias)"},
		{"/clear", "Clear conversation history"},
		{"/model", "Change AI model"},
		{"/persona", "Change AI persona"},
		{"/temp", "Adjust temperature (creativity)"},
		{"/save", "Save conversation to file"},
		{"/load", "Load a saved conversation"},
		{"/list", "List saved conversations"},
		{"/search <text>", "Search conversation history"},
		{"/stats", "Show conversation statistics"},
		{"/copy", "Copy last response (if available)"},
		{"/history", "Show last 10 messages"},
	}

	for _, cmd := range commands {
		color.Cyan(fmt.Sprintf("  %-20s ", cmd.cmd))
		color.White(cmd.desc + "\n")
	}
	fmt.Println()
}

// showHistory displays last N messages
func showHistory(state *ConversationState, n int) {
	if len(state.Messages) == 0 {
		color.Yellow("No messages in history.\n")
		return
	}

	color.Cyan(fmt.Sprintf("\n📜 Last %d Message(s):\n", n))

	start := len(state.Messages) - n
	if start < 0 {
		start = 0
	}

	for i := start; i < len(state.Messages); i++ {
		msg := state.Messages[i]
		if msg.Role == "user" {
			color.Cyan(fmt.Sprintf("\n👤 You:\n"))
		} else {
			color.Yellow(fmt.Sprintf("\n🤖 Assistant:\n"))
		}
		color.White(msg.Content + "\n")
	}
}

// streamResponse handles streaming responses
func streamResponse(reader io.Reader) (string, error) {
	var result strings.Builder
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			result.Write(buffer[:n])
			fmt.Print(string(buffer[:n]))
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return result.String(), nil
}

// printBannerEnhanced prints an enhanced welcome banner
func printBannerEnhanced(config *Config) {
	color.Cyan("╔══════════════════════════════════════════════════════════╗\n")
	color.Cyan("║         🚀 ShellSage - Enhanced AI Chat CLI v2.0          ║\n")
	color.Cyan("║      Powered by OpenRouter & Cutting-Edge LLMs           ║\n")
	color.Cyan("╚══════════════════════════════════════════════════════════╝\n\n")
	color.White(fmt.Sprintf("📌 Model: %s\n", config.Model))
	color.White("💡 Type /help for commands | /exit to quit\n\n")
}
