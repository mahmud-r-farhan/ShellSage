package tui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// PrintBanner outputs the premium ShellSage ASCII art banner
func PrintBanner(providerName, modelName, personaName string) {
	cyan := color.New(color.FgCyan, color.Bold)
	magenta := color.New(color.FgMagenta, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	white := color.New(color.FgHiWhite)
	hiCyan := color.New(color.FgHiCyan)
	dim := color.New(color.FgHiBlack)

	// ASCII Art Banner
	magenta.Println()
	cyan.Println(`  ███████╗██╗  ██╗███████╗██╗     ██╗     ███████╗ █████╗  ██████╗ ███████╗`)
	cyan.Println(`  ██╔════╝██║  ██║██╔════╝██║     ██║     ██╔════╝██╔══██╗██╔════╝ ██╔════╝`)
	cyan.Println(`  ███████╗███████║█████╗  ██║     ██║     ███████╗███████║██║  ███╗█████╗  `)
	cyan.Println(`  ╚════██║██╔══██║██╔══╝  ██║     ██║     ╚════██║██╔══██║██║   ██║██╔══╝  `)
	cyan.Println(`  ███████║██║  ██║███████╗███████╗███████╗███████║██║  ██║╚██████╔╝███████╗`)
	cyan.Println(`  ╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝`)
	dim.Println()
	hiCyan.Printf("  %-66s\n", "Autonomous AI Developer Platform  ·  v3.0  ·  Open Source")
	dim.Println("  " + strings.Repeat("─", 70))
	fmt.Println()

	// Info Row 1
	green.Print("  ⚡ Provider  : ")
	white.Printf("%-24s", providerName)
	green.Print("  🤖 Model : ")
	yellow.Println(modelName)

	// Info Row 2
	green.Print("  🎭 Persona   : ")
	white.Printf("%-24s", personaName)
	green.Print("  💡 Help  : ")
	yellow.Println("/help  |  /config  |  /exit")

	dim.Println("  " + strings.Repeat("─", 70))
	fmt.Println()
}

// PrintHelp outputs the full categorized command reference
func PrintHelp() {
	dim := color.New(color.FgHiBlack)
	dim.Println("\n" + strings.Repeat("─", 70))
	color.HiCyan("  📚  ShellSage Command Reference\n")
	dim.Println(strings.Repeat("─", 70))

	categories := []struct {
		Category string
		Commands []struct{ Cmd, Desc string }
	}{
		{
			Category: "🧠 Core & Configuration",
			Commands: []struct{ Cmd, Desc string }{
				{"/help", "Show this command reference guide"},
				{"/config", "Interactive provider & API key setup wizard"},
				{"/provider", "Switch LLM provider (OpenAI, Claude, Gemini, Groq, Ollama...)"},
				{"/model", "Switch model for the current provider"},
				{"/persona", "Switch AI persona | /persona create  → Custom wizard"},
				{"/temp", "Adjust creativity temperature (0.0 precise → 1.0 creative)"},
				{"/clear", "Clear conversation memory and start a fresh session"},
				{"/exit, /quit", "Auto-save session and exit ShellSage"},
			},
		},
		{
			Category: "🤖 Autonomous Agent & Developer Modes",
			Commands: []struct{ Cmd, Desc string }{
				{"/agent <goal>", "ReAct autonomous agent: filesystem, shell, web search & scrape"},
				{"/plan <task>", "Generate structured architectural implementation plan"},
				{"/debug <error>", "Deep root-cause diagnosis and automated patch generation"},
				{"/doc <file>", "Generate comprehensive Markdown documentation for code file"},
			},
		},
		{
			Category: "🛡️  Security & Vulnerability Analysis",
			Commands: []struct{ Cmd, Desc string }{
				{"/sec headers <url>", "Audit HTTP security headers (HSTS, CSP, X-Frame-Options, cookies)"},
				{"/sec ssl <domain>", "Inspect SSL/TLS certificate validity, expiry, and cipher suite"},
				{"/sec ports <host>", "Scan common developer & infrastructure service ports"},
				{"/sec sast [path]", "SAST code scan: leaked secrets, SQL injection, weak crypto"},
				{"/sec audit <target>", "Run full security assessment: headers + SSL + SAST combined"},
			},
		},
		{
			Category: "📋 Clipboard & Export",
			Commands: []struct{ Cmd, Desc string }{
				{"/copy", "Copy last response to OS clipboard"},
				{"/copy code", "Extract and copy only code blocks"},
				{"/copy all", "Copy full conversation transcript"},
				{"/export md [file]", "Export to GitHub-Flavored Markdown"},
				{"/export pdf [file]", "Export to clean printable PDF"},
				{"/export html [file]", "Export to dark-theme HTML file"},
			},
		},
		{
			Category: "🌿 Conversation Branching & Tree",
			Commands: []struct{ Cmd, Desc string }{
				{"/retry, /alt", "Generate an alternative response for the last turn"},
				{"/branch", "List and switch between conversation branches"},
				{"/tree", "Render ASCII conversation tree map"},
			},
		},
		{
			Category: "⏰ Task Queue & Scheduler",
			Commands: []struct{ Cmd, Desc string }{
				{"/queue add <task>", "Add a task to the background execution queue"},
				{"/queue list", "List all queued tasks and their status"},
				{"/queue run", "Execute the next pending task"},
				{"/schedule at <HH:MM> <task>", "Run task at exact local machine time"},
				{"/schedule in <dur> <task>", "Run task after relative delay (e.g. 30m, 2h)"},
				{"/schedule list", "List scheduled jobs"},
			},
		},
		{
			Category: "💾 History, Analytics & Sessions",
			Commands: []struct{ Cmd, Desc string }{
				{"/save [filename]", "Save conversation session to file"},
				{"/load [filename]", "Load a saved session"},
				{"/list", "List all saved conversation files"},
				{"/history [N]", "Show last N messages"},
				{"/search <term>", "Search through conversation history"},
				{"/stats, /analytics", "Token usage, cost estimate, and session metrics"},
			},
		},
	}

	for _, cat := range categories {
		fmt.Println()
		color.New(color.FgGreen, color.Bold).Printf("  %s\n", cat.Category)
		for _, cmd := range cat.Commands {
			color.HiCyan(fmt.Sprintf("    %-32s", cmd.Cmd))
			color.White("  %s\n", cmd.Desc)
		}
	}

	fmt.Println()
	dim.Println(strings.Repeat("─", 70))
	color.HiBlack("  CLI Flags: --agent  --plan  --debug  --doc  --config  --version\n")
	dim.Println(strings.Repeat("─", 70))
	fmt.Println()
}

// PrintSeparator outputs a stylish divider line
func PrintSeparator() {
	color.HiBlack(strings.Repeat("─", 70) + "\n")
}
