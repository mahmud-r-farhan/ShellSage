package tui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// PrintBanner outputs the premium ShellSage banner
func PrintBanner(providerName, modelName, personaName string) {
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgHiWhite)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("╔══════════════════════════════════════════════════════════════════╗")
	cyan.Println("║               🚀  S H E L L S A G E   v3.0  🚀                   ║")
	cyan.Println("║       The Autonomous AI Terminal Assistant for Developers        ║")
	cyan.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	green.Print("  ⚡ Provider: ")
	white.Printf("%-18s ", providerName)
	green.Print("🤖 Model: ")
	yellow.Printf("%s\n", modelName)

	green.Print("  🎭 Persona:  ")
	white.Printf("%-18s ", personaName)
	green.Print("💡 Help:  ")
	yellow.Println("Type /help for commands, /config to setup")
	fmt.Println()
}

// PrintHelp outputs the categorized extended command reference
func PrintHelp() {
	color.Cyan("\n📚 ━━━━━━━━━━━━ ShellSage Command Reference ━━━━━━━━━━━━\n\n")

	categories := []struct {
		Category string
		Commands []struct{ Cmd, Desc string }
	}{
		{
			Category: "🧠 Core & Navigation",
			Commands: []struct{ Cmd, Desc string }{
				{"/help", "Show this interactive command guide"},
				{"/config", "Launch interactive Configuration & Provider Setup Wizard"},
				{"/provider", "Switch active LLM provider (OpenAI, Claude, Gemini, Groq, Ollama...)"},
				{"/model", "Switch active model for current provider"},
				{"/persona", "Switch persona or create a custom persona (/persona create)"},
				{"/temp", "Adjust LLM temperature / creativity (0.0 - 1.0)"},
				{"/clear", "Clear current conversation memory"},
				{"/exit, /quit", "Save state and exit ShellSage"},
			},
		},
		{
			Category: "📋 Clipboard & Export",
			Commands: []struct{ Cmd, Desc string }{
				{"/copy", "Copy last assistant response to OS clipboard"},
				{"/copy code", "Extract and copy ONLY code blocks from last response"},
				{"/copy all", "Copy entire active conversation transcript to clipboard"},
				{"/export md [file]", "Export conversation to GitHub-Flavored Markdown"},
				{"/export pdf [file]", "Export conversation to clean PDF report"},
				{"/export html [file]", "Export conversation to dark-mode HTML file"},
			},
		},
		{
			Category: "🌿 Conversation Branching & Tree",
			Commands: []struct{ Cmd, Desc string }{
				{"/retry, /alt", "Generate an alternative response for the last turn"},
				{"/branch", "List and switch between conversation branches"},
				{"/tree", "Render visual ASCII conversation tree"},
			},
		},
		{
			Category: "🤖 Autonomous Agent & Developer Tools",
			Commands: []struct{ Cmd, Desc string }{
				{"/agent <goal>", "Execute goal autonomously with local tools & web search"},
				{"/plan <task>", "Generate comprehensive architectural implementation plan"},
				{"/debug <error>", "Perform deep root-cause diagnosis & generate fix"},
				{"/doc <file>", "Generate comprehensive documentation / README for code"},
				{"/search <text>", "Search through conversation history"},
				{"/stats, /analytics", "Display token consumption, costs, and session metrics"},
			},
		},
		{
			Category: "⏰ Task Queue & Local Time Scheduler",
			Commands: []struct{ Cmd, Desc string }{
				{"/queue add <task>", "Add a task to background execution queue"},
				{"/queue list", "List all tasks in the queue"},
				{"/queue run", "Execute next pending task in queue"},
				{"/schedule at <HH:MM> <task>", "Schedule task at local machine time (e.g. 15:30)"},
				{"/schedule in <duration> <task>", "Schedule task in relative time (e.g. 10m, 1h)"},
				{"/schedule list", "List all scheduled jobs"},
			},
		},
		{
			Category: "💾 History & Sessions",
			Commands: []struct{ Cmd, Desc string }{
				{"/save [filename]", "Save current conversation session to file"},
				{"/load [filename]", "Load a previously saved session"},
				{"/list", "List all saved conversation files"},
				{"/history [N]", "Display the last N messages"},
			},
		},
	}

	for _, cat := range categories {
		color.Green("  " + cat.Category + "\n")
		for _, cmd := range cat.Commands {
			color.Cyan(fmt.Sprintf("    %-30s", cmd.Cmd))
			color.White(" - " + cmd.Desc + "\n")
		}
		fmt.Println()
	}
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
}

// PrintSeparator outputs a stylish divider line
func PrintSeparator() {
	color.HiBlack(strings.Repeat("─", 65) + "\n")
}
