package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"shellsage/internal/branch"
	"shellsage/internal/config"
	"shellsage/internal/persona"
)

// SelectProviderMenu prompts the user to select an active LLM provider
func SelectProviderMenu(current config.ProviderType) config.ProviderType {
	reader := bufio.NewReader(os.Stdin)

	providers := []struct {
		Type config.ProviderType
		Name string
		Desc string
	}{
		{config.ProviderOpenRouter, "OpenRouter", "Unified hub for 200+ models (Free & Paid)"},
		{config.ProviderOpenAI, "OpenAI", "Direct API for GPT-4o, o1, o3-mini"},
		{config.ProviderAnthropic, "Anthropic", "Direct API for Claude 3.5 & 3.7 Sonnet/Haiku"},
		{config.ProviderGemini, "Google Gemini", "Gemini 2.0 Flash / Pro"},
		{config.ProviderGroq, "Groq Cloud", "Ultra-fast inference (Llama 3.3, Mixtral, DeepSeek)"},
		{config.ProviderDeepSeek, "DeepSeek", "Direct API for DeepSeek-V3 & DeepSeek-R1"},
		{config.ProviderOllama, "Ollama (Local)", "Run offline local models (llama3, mistral, qwen)"},
		{config.ProviderCustom, "Custom / Self-Hosted", "Any OpenAI-compatible server (vLLM, LM Studio)"},
	}

	color.Cyan("\n🌐 ━━ Select LLM Provider ━━\n")
	for i, p := range providers {
		activeTag := ""
		if p.Type == current {
			activeTag = color.HiGreenString(" [CURRENT]")
		}
		color.White("  [%d] %-22s %s%s\n", i+1, p.Name, color.HiBlackString("- "+p.Desc), activeTag)
	}
	color.White("  [0] Keep current (%s)\n", current)

	color.Cyan("\nEnter choice (0-%d): ", len(providers))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" || input == "0" {
		return current
	}

	idx, err := strconv.Atoi(input)
	if err == nil && idx >= 1 && idx <= len(providers) {
		return providers[idx-1].Type
	}

	return current
}

// SelectModelMenu displays available models for selection
func SelectModelMenu(currentModel string, availableModels []string) string {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan("\n🤖 ━━ Select AI Model ━━\n")
	for i, model := range availableModels {
		activeTag := ""
		if model == currentModel {
			activeTag = color.HiGreenString(" [CURRENT]")
		}
		color.White("  [%d] %s%s\n", i+1, model, activeTag)
	}
	color.White("  [0] Keep current (%s)\n", currentModel)
	color.White("  [c] Enter custom model name\n")

	color.Cyan("\nEnter choice: ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "" || choice == "0" {
		return currentModel
	}

	if strings.ToLower(choice) == "c" {
		color.Cyan("Enter model identifier: ")
		custom, _ := reader.ReadString('\n')
		custom = strings.TrimSpace(custom)
		if custom != "" {
			return custom
		}
		return currentModel
	}

	idx, err := strconv.Atoi(choice)
	if err == nil && idx >= 1 && idx <= len(availableModels) {
		return availableModels[idx-1]
	}

	return choice
}

// SelectPersonaMenu displays personas including custom ones
func SelectPersonaMenu(mgr *persona.PersonaManager, currentID string) persona.Persona {
	reader := bufio.NewReader(os.Stdin)
	list := mgr.ListSorted()

	color.Cyan("\n🎭 ━━ Select AI Persona ━━\n")
	for i, p := range list {
		badge := ""
		if p.IsCustom {
			badge = color.HiYellowString("[CUSTOM] ")
		}
		activeTag := ""
		if p.ID == currentID {
			activeTag = color.HiGreenString(" [CURRENT]")
		}
		color.White("  [%d] %s%-25s %s%s\n", i+1, badge, p.Name, color.HiBlackString("("+p.Description+")"), activeTag)
	}
	color.White("  [0] Keep current\n")
	color.White("  [+] Create new custom persona\n")

	color.Cyan("\nEnter choice: ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "" || choice == "0" {
		p, _ := mgr.Get(currentID)
		return p
	}

	if choice == "+" || strings.ToLower(choice) == "create" {
		return CreatePersonaWizard(mgr)
	}

	idx, err := strconv.Atoi(choice)
	if err == nil && idx >= 1 && idx <= len(list) {
		return list[idx-1]
	}

	if p, ok := mgr.Get(choice); ok {
		return p
	}

	p, _ := mgr.Get(currentID)
	return p
}

// CreatePersonaWizard guides the user to create a new custom persona
func CreatePersonaWizard(mgr *persona.PersonaManager) persona.Persona {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan("\n✨ ━━ Custom Persona Creation Wizard ━━\n")

	color.White("Enter Persona Name (e.g. Rust Microservices Architect): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Custom Specialist"
	}

	color.White("Enter Short Description: ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)
	if desc == "" {
		desc = "Custom user-defined persona"
	}

	color.White("Enter System Prompt (Instructions for AI behavior):\n> ")
	prompt, _ := reader.ReadString('\n')
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = "You are a helpful and specialized AI assistant."
	}

	p := persona.Persona{
		ID:          strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Name:        name,
		Description: desc,
		Prompt:      prompt,
	}

	if err := mgr.SaveCustom(p); err != nil {
		color.Red("Failed to save custom persona: %v\n", err)
	} else {
		color.Green("✅ Custom persona '%s' created and saved!\n", name)
	}

	return p
}

// SelectBranchMenu displays branches for branching tree navigation
func SelectBranchMenu(tree *branch.ConversationTree) bool {
	branches := tree.ListBranches()
	if len(branches) <= 1 {
		color.Yellow("\n🌱 Only 1 branch exists in the current conversation.\n")
		return false
	}

	reader := bufio.NewReader(os.Stdin)
	color.Cyan("\n🌿 ━━ Conversation Branches ━━\n")
	for _, b := range branches {
		activeTag := ""
		if b.IsActive {
			activeTag = color.HiGreenString(" [ACTIVE]")
		}
		color.White("  [%d] Branch %d (%d msgs) - \"%s\"%s\n", b.Index, b.Index, b.MessageCount, b.LastSnippet, activeTag)
	}

	color.Cyan("\nSelect branch number to switch: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err == nil && idx >= 1 && idx <= len(branches) {
		selected := branches[idx-1]
		tree.SwitchBranch(selected.LeafNodeID)
		color.Green("✅ Switched to Branch %d!\n", idx)
		return true
	}

	return false
}

// AdjustTemperatureMenu prompts to adjust temperature
func AdjustTemperatureMenu(current float64) float64 {
	reader := bufio.NewReader(os.Stdin)
	color.Cyan("\n🌡️  Current Temperature: %.2f (0.0 = Precise & Deterministic, 1.0 = Creative)\n", current)
	color.Cyan("Enter new value (0.0 - 1.0) or press Enter to keep: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current
	}

	var val float64
	_, err := fmt.Sscanf(input, "%f", &val)
	if err != nil || val < 0.0 || val > 2.0 {
		color.Red("Invalid temperature value. Keeping current: %.2f\n", current)
		return current
	}

	return val
}
