package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"shellsage/internal/agent"
	"shellsage/internal/branch"
	"shellsage/internal/clipboard"
	"shellsage/internal/config"
	"shellsage/internal/export"
	"shellsage/internal/history"
	"shellsage/internal/persona"
	"shellsage/internal/provider"
	"shellsage/internal/scheduler"
	"shellsage/internal/security"
	"shellsage/internal/tools"
	"shellsage/internal/tui"
)

const Version = "3.0.0"

func main() {
	// Parse CLI flags and subcommands
	versionFlag := flag.Bool("version", false, "Print ShellSage version")
	configFlag := flag.Bool("config", false, "Launch interactive configuration setup wizard")
	agentFlag := flag.String("agent", "", "Run autonomous agent with a specified goal and exit")
	planFlag := flag.String("plan", "", "Generate an implementation plan for a requirement and exit")
	debugFlag := flag.String("debug", "", "Diagnose an error message or log file and exit")
	docFlag := flag.String("doc", "", "Generate documentation for a specified file path and exit")
	auditFlag := flag.String("audit", "", "Run security and vulnerability audit on target URL or codebase and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("ShellSage AI CLI v%s\n", Version)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		color.Yellow("⚠️ Notice: %v", err)
	}

	if *configFlag || (len(flag.Args()) > 0 && flag.Args()[0] == "config") {
		_ = tui.RunConfigWizard(cfg)
		return
	}

	// Initialize Persona Manager & Active Persona
	personaMgr := persona.NewPersonaManager()
	activePersona, _ := personaMgr.Get("general")

	// Initialize Provider
	pClient, err := provider.NewProvider(cfg)
	if err != nil {
		color.Red("❌ Error initializing provider: %v", err)
		color.Yellow("Launching setup wizard to configure provider...")
		_ = tui.RunConfigWizard(cfg)
		pClient, _ = provider.NewProvider(cfg)
	}

	// Active Model
	pCfg, _ := cfg.GetActiveProviderConfig()
	activeModel := pCfg.Model
	activeTemp := cfg.DefaultTemp

	// Initialize Tools, Agent, Queue, Scheduler
	toolReg := tools.NewToolRegistry()
	autonomousAgent := agent.NewAgent(pClient, toolReg, activeModel, activeTemp)
	taskQueue := scheduler.NewTaskQueue(autonomousAgent)
	taskScheduler := scheduler.NewScheduler(autonomousAgent)

	// Non-interactive subcommands
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *agentFlag != "" {
		runAgentDirect(ctx, autonomousAgent, *agentFlag)
		return
	}
	if *planFlag != "" {
		plan, err := autonomousAgent.GeneratePlan(ctx, *planFlag)
		if err != nil {
			color.Red("Error generating plan: %v", err)
		} else {
			fmt.Println(plan)
		}
		return
	}
	if *debugFlag != "" {
		diag, err := autonomousAgent.DiagnoseError(ctx, *debugFlag)
		if err != nil {
			color.Red("Error diagnosing issue: %v", err)
		} else {
			fmt.Println(diag)
		}
		return
	}
	if *docFlag != "" {
		content, _ := os.ReadFile(*docFlag)
		docs, err := autonomousAgent.GenerateDocumentation(ctx, *docFlag, string(content))
		if err != nil {
			color.Red("Error generating documentation: %v", err)
		} else {
			fmt.Println(docs)
		}
		return
	}
	if *auditFlag != "" {
		fmt.Print(security.RunFullAudit(ctx, *auditFlag))
		return
	}

	// Start background scheduler listener
	taskScheduler.Start(ctx, func(j *scheduler.ScheduledJob) {
		color.HiMagenta("\n🔔 [SCHEDULER NOTIFICATION] Task '%s' completed at %s!\n", j.Goal, time.Now().Format("15:04:05"))
		if j.Error != "" {
			color.Red("   Error: %s\n", j.Error)
		} else {
			color.Green("   Result: %s\n", truncateString(j.Result, 120))
		}
		color.Cyan("👤 You: ")
	})
	defer taskScheduler.Stop()

	// Print Welcome Banner
	tui.PrintBanner(string(cfg.ActiveProvider), activeModel, activePersona.Name)

	// Initialize Conversation Tree & Session Analytics
	tree := branch.NewConversationTree(activePersona.ID)
	sessionAnalytics := history.NewSessionAnalytics()

	// Handle Graceful Termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		if cfg.AutoSaveHistory && len(tree.Nodes) > 0 {
			_, _ = history.SaveTree(tree, "")
		}
		color.Green("\n✨ Goodbye! Session ended gracefully.\n")
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	// Main Interactive REPL Loop
	for {
		color.Cyan("\n👤 You: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}

		// Handle Slash Commands
		if strings.HasPrefix(userInput, "/") {
			parts := strings.Fields(userInput)
			cmd := strings.ToLower(parts[0])

			switch cmd {
			case "/exit", "/quit":
				if cfg.AutoSaveHistory && len(tree.Nodes) > 0 {
					savedPath, _ := history.SaveTree(tree, "")
					if savedPath != "" {
						color.HiBlack("💾 Session auto-saved to %s\n", savedPath)
					}
				}
				fmt.Print(sessionAnalytics.FormatSummary(activePersona.Name, activeModel, tree))
				color.Green("✨ Goodbye! Thanks for using ShellSage.\n\n")
				return

			case "/help":
				tui.PrintHelp()
				continue

			case "/config":
				if err := tui.RunConfigWizard(cfg); err == nil {
					pClient, _ = provider.NewProvider(cfg)
					pCfg, _ = cfg.GetActiveProviderConfig()
					activeModel = pCfg.Model
					autonomousAgent = agent.NewAgent(pClient, toolReg, activeModel, activeTemp)
					color.Green("✅ Provider & Model updated to %s (%s)\n", cfg.ActiveProvider, activeModel)
				}
				continue

			case "/provider":
				newProv := tui.SelectProviderMenu(cfg.ActiveProvider)
				if newProv != cfg.ActiveProvider {
					cfg.ActiveProvider = newProv
					pCfg, _ = cfg.GetActiveProviderConfig()
					activeModel = pCfg.Model
					pClient, err = provider.NewProvider(cfg)
					if err != nil {
						color.Red("Error switching provider: %v", err)
					} else {
						autonomousAgent = agent.NewAgent(pClient, toolReg, activeModel, activeTemp)
						color.Green("✅ Switched provider to %s (Model: %s)\n", newProv, activeModel)
					}
				}
				continue

			case "/model":
				newModel := tui.SelectModelMenu(activeModel, pClient.ListAvailableModels())
				if newModel != activeModel {
					activeModel = newModel
					pCfg.Model = newModel
					cfg.Providers[cfg.ActiveProvider] = pCfg
					autonomousAgent = agent.NewAgent(pClient, toolReg, activeModel, activeTemp)
					color.Green("✅ Model switched to: %s\n", activeModel)
				}
				continue

			case "/persona":
				if len(parts) > 1 && strings.ToLower(parts[1]) == "create" {
					activePersona = tui.CreatePersonaWizard(personaMgr)
				} else {
					activePersona = tui.SelectPersonaMenu(personaMgr, activePersona.ID)
				}
				tree.PersonaID = activePersona.ID
				color.Green("✅ Active Persona set to: %s\n", activePersona.Name)
				continue

			case "/temp":
				activeTemp = tui.AdjustTemperatureMenu(activeTemp)
				autonomousAgent = agent.NewAgent(pClient, toolReg, activeModel, activeTemp)
				color.Green("✅ Temperature adjusted to: %.2f\n", activeTemp)
				continue

			case "/clear":
				if cfg.AutoSaveHistory && len(tree.Nodes) > 0 {
					_, _ = history.SaveTree(tree, "")
				}
				tree = branch.NewConversationTree(activePersona.ID)
				color.Green("✅ Conversation history cleared! Started fresh tree.\n")
				continue

			case "/copy":
				handleCopyCommand(parts, tree)
				continue

			case "/export":
				handleExportCommand(parts, tree)
				continue

			case "/retry", "/alt":
				if userNode, ok := tree.RewindForAlternative(); ok {
					color.Yellow("🔄 Generating alternative response for: \"%s\"...\n", truncateString(userNode.Content, 40))
					generateAndStreamResponse(ctx, pClient, tree, activePersona, activeModel, activeTemp, cfg.StreamResponses, sessionAnalytics)
				} else {
					color.Yellow("⚠️ Cannot rewind: no preceding user turn found to regenerate.\n")
				}
				continue

			case "/branch":
				tui.SelectBranchMenu(tree)
				continue

			case "/tree":
				fmt.Print(tree.RenderTreeAscii())
				continue

			case "/stats", "/analytics":
				fmt.Print(sessionAnalytics.FormatSummary(activePersona.Name, activeModel, tree))
				continue

			case "/agent":
				if len(parts) > 1 {
					goal := strings.Join(parts[1:], " ")
					runAgentDirect(ctx, autonomousAgent, goal)
				} else {
					color.Yellow("Usage: /agent <goal description>\n")
				}
				continue

			case "/plan":
				if len(parts) > 1 {
					reqGoal := strings.Join(parts[1:], " ")
					color.Yellow("\n📐 Generating Architectural Plan...\n")
					plan, err := autonomousAgent.GeneratePlan(ctx, reqGoal)
					if err != nil {
						color.Red("Error: %v\n", err)
					} else {
						fmt.Println(plan)
						tree.AddMessage("user", "/plan "+reqGoal, activeModel, provider.TokenUsage{})
						tree.AddMessage("assistant", plan, activeModel, provider.TokenUsage{})
					}
				} else {
					color.Yellow("Usage: /plan <feature or system requirement>\n")
				}
				continue

			case "/debug":
				if len(parts) > 1 {
					errText := strings.Join(parts[1:], " ")
					color.Yellow("\n🔍 Analyzing Error and Root Cause...\n")
					diag, err := autonomousAgent.DiagnoseError(ctx, errText)
					if err != nil {
						color.Red("Error: %v\n", err)
					} else {
						fmt.Println(diag)
						tree.AddMessage("user", "/debug "+errText, activeModel, provider.TokenUsage{})
						tree.AddMessage("assistant", diag, activeModel, provider.TokenUsage{})
					}
				} else {
					color.Yellow("Usage: /debug <error message or stack trace>\n")
				}
				continue

			case "/doc":
				if len(parts) > 1 {
					path := parts[1]
					fileBytes, err := os.ReadFile(path)
					if err != nil {
						color.Red("Error reading file '%s': %v\n", path, err)
						continue
					}
					color.Yellow("\n📖 Generating Documentation for %s...\n", path)
					docs, err := autonomousAgent.GenerateDocumentation(ctx, path, string(fileBytes))
					if err != nil {
						color.Red("Error: %v\n", err)
					} else {
						fmt.Println(docs)
					}
				} else {
					color.Yellow("Usage: /doc <file path>\n")
				}
				continue

			case "/queue":
				handleQueueCommand(ctx, parts, taskQueue)
				continue

			case "/schedule":
				handleScheduleCommand(parts, taskScheduler)
				continue

			case "/sec":
				handleSecCommand(ctx, parts)
				continue

			case "/audit":
				target := "."
				if len(parts) > 1 {
					target = parts[1]
				}
				fmt.Print(security.RunFullAudit(ctx, target))
				continue

			case "/save":
				fname := ""
				if len(parts) > 1 {
					fname = parts[1]
				}
				savedPath, err := history.SaveTree(tree, fname)
				if err != nil {
					color.Red("Error saving: %v\n", err)
				} else {
					color.Green("✅ Conversation saved to %s\n", savedPath)
				}
				continue

			case "/load":
				if len(parts) > 1 {
					loaded, err := history.LoadTree(parts[1])
					if err != nil {
						color.Red("Error loading: %v\n", err)
					} else {
						tree = loaded
						color.Green("✅ Loaded conversation '%s' (%d turns)\n", tree.Title, len(tree.GetActivePath()))
					}
				} else {
					showSavedConversations()
					color.Cyan("Enter filename to load: ")
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(input)
					if input != "" {
						loaded, err := history.LoadTree(input)
						if err != nil {
							color.Red("Error loading: %v\n", err)
						} else {
							tree = loaded
							color.Green("✅ Loaded conversation '%s' (%d turns)\n", tree.Title, len(tree.GetActivePath()))
						}
					}
				}
				continue

			case "/list":
				showSavedConversations()
				continue

			case "/history":
				limit := 10
				if len(parts) > 1 {
					if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
						limit = n
					}
				}
				showActiveHistory(tree, limit)
				continue

			case "/search":
				if len(parts) > 1 {
					query := strings.Join(parts[1:], " ")
					searchActiveTree(tree, query)
				} else {
					color.Yellow("Usage: /search <term>\n")
				}
				continue

			default:
				color.Yellow("❌ Unknown command: %s (Type /help for command list)\n", parts[0])
				continue
			}
		}

		// Regular User Chat Message
		tree.AddMessage("user", userInput, activeModel, provider.TokenUsage{})
		generateAndStreamResponse(ctx, pClient, tree, activePersona, activeModel, activeTemp, cfg.StreamResponses, sessionAnalytics)
	}
}

func generateAndStreamResponse(ctx context.Context, p provider.Provider, tree *branch.ConversationTree, pActive persona.Persona, model string, temp float64, stream bool, analytics *history.SessionAnalytics) {
	// Build messages with system persona prompt
	messages := []provider.Message{
		{Role: "system", Content: pActive.Prompt},
	}
	messages = append(messages, tree.GetActiveMessages()...)

	req := &provider.ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temp,
		Stream:      stream,
	}

	color.Yellow("\n🤖 Assistant: ")

	if stream {
		resp, err := p.Stream(ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			color.Red("\n❌ Stream error: %v\n", err)
			return
		}
		fmt.Println()

		// Save response node to conversation tree
		tree.AddMessage("assistant", resp.Content, model, resp.Usage)
		analytics.RecordTurn(model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

		color.HiBlack("[Tokens: +%d in, +%d out | Total: %d]\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, analytics.TotalTokens)
	} else {
		resp, err := p.Chat(ctx, req)
		if err != nil {
			color.Red("\n❌ API error: %v\n", err)
			return
		}

		fmt.Println(resp.Content)
		tree.AddMessage("assistant", resp.Content, model, resp.Usage)
		analytics.RecordTurn(model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

		color.HiBlack("\n[Tokens: +%d in, +%d out | Total: %d]\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, analytics.TotalTokens)
	}
}

func handleCopyCommand(parts []string, tree *branch.ConversationTree) {
	lastNode := tree.GetLastMessage()
	if lastNode == nil {
		color.Yellow("⚠️ Conversation is empty, nothing to copy.\n")
		return
	}

	sub := ""
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}

	switch sub {
	case "code":
		if lastNode.Role != "assistant" {
			color.Yellow("⚠️ Last message is not an assistant response.\n")
			return
		}
		code, err := clipboard.CopyCodeOnly(lastNode.Content)
		if err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Copied code block(s) to system clipboard!\n")
			color.HiBlack("--- Copied Snippet ---\n%s\n----------------------\n", truncateString(code, 200))
		}

	case "all":
		md := export.ExportToMarkdown(tree)
		if err := clipboard.CopyText(md); err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Copied entire conversation transcript to system clipboard!\n")
		}

	default:
		// Default: copy last assistant response text
		if lastNode.Role != "assistant" {
			color.Yellow("⚠️ Last message is not an assistant response.\n")
			return
		}
		if err := clipboard.CopyText(lastNode.Content); err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Last assistant response copied to clipboard!\n")
		}
	}
}

func handleExportCommand(parts []string, tree *branch.ConversationTree) {
	if len(parts) < 2 {
		color.Yellow("Usage: /export <md|pdf|html|json> [optional_filename]\n")
		return
	}

	fmtType := export.ExportFormat(strings.ToLower(parts[1]))
	fname := ""
	if len(parts) > 2 {
		fname = parts[2]
	}

	outPath, err := export.ExportConversation(tree, fmtType, fname)
	if err != nil {
		color.Red("❌ Export failed: %v\n", err)
	} else {
		color.Green("✅ Conversation successfully exported to %s\n", outPath)
	}
}

func handleQueueCommand(ctx context.Context, parts []string, q *scheduler.TaskQueue) {
	if len(parts) < 2 {
		color.Yellow("Usage: /queue <add|list|run> [arguments]\n")
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "add":
		if len(parts) < 3 {
			color.Yellow("Usage: /queue add <task description>\n")
			return
		}
		desc := strings.Join(parts[2:], " ")
		task := q.Add(desc)
		color.Green("✅ Added task [%s]: %s\n", task.ID, task.Description)

	case "list":
		tasks := q.List()
		if len(tasks) == 0 {
			color.Yellow("Task queue is empty.\n")
			return
		}
		color.Cyan("\n📋 ━━ Task Queue (%d tasks) ━━\n", len(tasks))
		for i, t := range tasks {
			statusColor := color.YellowString
			if t.Status == scheduler.StatusCompleted {
				statusColor = color.GreenString
			} else if t.Status == scheduler.StatusFailed {
				statusColor = color.RedString
			}
			color.White("  [%d] %-10s %-12s: %s\n", i+1, t.ID, statusColor(string(t.Status)), t.Description)
		}
		fmt.Println()

	case "run":
		color.Yellow("⚙️ Processing next queued task...\n")
		listener := &agent.DefaultListener{
			ToolCallHandler: func(name, args string) {
				color.HiCyan("  🔧 [Tool: %s] args: %s", name, truncateString(args, 80))
			},
		}
		task, err := q.ExecuteNext(ctx, listener)
		if err != nil {
			color.Red("❌ Task failed: %v\n", err)
		} else {
			color.Green("\n✅ Task [%s] Completed!\nResult:\n%s\n", task.ID, task.Result)
		}
	}
}

func handleScheduleCommand(parts []string, s *scheduler.Scheduler) {
	if len(parts) < 2 {
		color.Yellow("Usage: /schedule <at|in|list> [arguments]\nExample: /schedule at 15:30 Check status\n")
		return
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "at":
		if len(parts) < 4 {
			color.Yellow("Usage: /schedule at <HH:MM> <goal>\nExample: /schedule at 16:00 Run integration tests\n")
			return
		}
		timeParts := strings.Split(parts[2], ":")
		if len(timeParts) != 2 {
			color.Red("Invalid time format. Use HH:MM (e.g. 14:30)\n")
			return
		}
		hour, _ := strconv.Atoi(timeParts[0])
		min, _ := strconv.Atoi(timeParts[1])
		goal := strings.Join(parts[3:], " ")

		job, err := s.ScheduleAt(hour, min, goal)
		if err != nil {
			color.Red("Error scheduling: %v\n", err)
		} else {
			color.Green("✅ Scheduled task [%s] for %s (Local Time)!\n", job.ID, job.TargetTime.Format("Jan 02 15:04:05"))
		}

	case "in":
		if len(parts) < 4 {
			color.Yellow("Usage: /schedule in <duration> <goal>\nExample: /schedule in 30m Check server health\n")
			return
		}
		dur, err := time.ParseDuration(parts[2])
		if err != nil {
			color.Red("Invalid duration format (e.g. 10m, 1h, 45s): %v\n", err)
			return
		}
		goal := strings.Join(parts[3:], " ")
		job, err := s.ScheduleIn(dur, goal)
		if err != nil {
			color.Red("Error scheduling: %v\n", err)
		} else {
			color.Green("✅ Scheduled task [%s] to run in %v (at %s)!\n", job.ID, dur, job.TargetTime.Format("15:04:05"))
		}

	case "list":
		jobs := s.ListJobs()
		if len(jobs) == 0 {
			color.Yellow("No scheduled jobs found.\n")
			return
		}
		color.Cyan("\n⏰ ━━ Scheduled Jobs ━━\n")
		for _, j := range jobs {
			execStatus := color.YellowString("PENDING")
			if j.Executed {
				if j.Error != "" {
					execStatus = color.RedString("FAILED")
				} else {
					execStatus = color.GreenString("EXECUTED")
				}
			}
			color.White("  [%s] Target: %s | Status: %s\n    Goal: %s\n", j.ID, j.TargetTime.Format("15:04:05"), execStatus, j.Goal)
		}
		fmt.Println()
	}
}

func runAgentDirect(ctx context.Context, ag *agent.Agent, goal string) {
	color.Cyan("\n🤖 ━━ Autonomous Agent Initiated ━━\n")
	color.White("Goal: %s\n\n", goal)

	listener := &agent.DefaultListener{
		ToolCallHandler: func(name, args string) {
			color.HiCyan("  🔧 Executing tool [%s] with args: %s\n", name, truncateString(args, 100))
		},
		ToolResultHandler: func(name, res string) {
			color.HiBlack("     ↳ Result (%d chars): %s\n", len(res), truncateString(res, 80))
		},
		FinalAnswerHandler: func(ans string) {
			color.Green("\n🎯 Final Agent Response:\n\n")
			fmt.Println(ans)
		},
	}

	_, err := ag.Run(ctx, goal, listener)
	if err != nil {
		color.Red("\n❌ Agent encountered an error: %v\n", err)
	}
}

func showSavedConversations() {
	list := history.ListSavedConversations()
	if len(list) == 0 {
		color.Yellow("No saved conversations found in 'conversations/' directory.\n")
		return
	}

	color.Cyan("\n💾 ━━ Saved Conversations ━━\n")
	for i, c := range list {
		color.White("  [%d] %-30s %s (%d turns, %d tokens)\n", i+1, c.Filename, color.HiBlackString("["+c.Title+"]"), c.MessageCount, c.TotalTokens)
	}
	fmt.Println()
}

func showActiveHistory(tree *branch.ConversationTree, limit int) {
	path := tree.GetActivePath()
	if len(path) == 0 {
		color.Yellow("No messages in active conversation path.\n")
		return
	}

	color.Cyan("\n📜 ━━ Last %d Messages ━━\n", limit)
	start := len(path) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(path); i++ {
		node := path[i]
		if node.Role == "user" {
			color.Cyan("\n👤 You (%s):\n", node.Timestamp.Format("15:04:05"))
		} else {
			color.Yellow("\n🤖 Assistant (%s):\n", node.Timestamp.Format("15:04:05"))
		}
		color.White(node.Content + "\n")
	}
}

func searchActiveTree(tree *branch.ConversationTree, query string) {
	path := tree.GetActivePath()
	queryLower := strings.ToLower(query)
	found := 0

	color.Cyan("\n🔍 Search Results for \"%s\":\n", query)
	for i, node := range path {
		if strings.Contains(strings.ToLower(node.Content), queryLower) {
			found++
			color.Yellow("\n[Message %d - %s]:\n", i+1, node.Role)
			color.White(node.Content + "\n")
		}
	}

	if found == 0 {
		color.Yellow("No matching messages found.\n")
	} else {
		color.Green("\n✅ Found %d matching message(s)\n", found)
	}
}

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func handleSecCommand(ctx context.Context, parts []string) {
	if len(parts) < 2 {
		color.Yellow("Usage: /sec <headers|ssl|ports|sast|audit> [target]\n")
		color.White("  /sec headers <url>      - Audit HTTP security headers & cookie flags\n")
		color.White("  /sec ssl <domain>       - Inspect SSL/TLS certificate & cipher suite\n")
		color.White("  /sec ports <host>       - Scan common dev & infrastructure service ports\n")
		color.White("  /sec sast [path]        - Scan code for leaked secrets & vulnerabilities\n")
		color.White("  /sec audit <target>     - Run full combined security posture audit\n")
		return
	}

	sub := strings.ToLower(parts[1])
	target := "."
	if len(parts) > 2 {
		target = parts[2]
	}

	switch sub {
	case "headers":
		if len(parts) < 3 {
			color.Yellow("Usage: /sec headers <url>\nExample: /sec headers https://example.com\n")
			return
		}
		color.Cyan("\n🌐 Auditing Security Headers for %s...\n\n", target)
		res, err := security.AuditSecurityHeaders(ctx, target)
		if err != nil {
			color.Red("❌ Audit failed: %v\n", err)
			return
		}
		color.Green("📊 Grade: %s  (Security Score: %d/100)\n\n", res.Grade, res.Score)
		if len(res.PresentHeaders) > 0 {
			color.White("  ✅ Present Security Headers:\n")
			for k, v := range res.PresentHeaders {
				color.HiGreen("     • %s: %s\n", k, truncateString(v, 60))
			}
		}
		if len(res.MissingHeaders) > 0 {
			color.White("\n  ⚠️  Missing Headers:\n")
			for _, m := range res.MissingHeaders {
				color.HiYellow("     • %s\n", m)
			}
		}
		if len(res.Warnings) > 0 {
			color.White("\n  ⚠️  Information Disclosure Warnings:\n")
			for _, w := range res.Warnings {
				color.HiRed("     • %s\n", w)
			}
		}
		if len(res.Cookies) > 0 {
			color.White("\n  🍪 Cookies Security Analysis:\n")
			for _, c := range res.Cookies {
				color.Cyan("     • %s\n", c)
			}
		}
		fmt.Println()

	case "ssl":
		if len(parts) < 3 {
			color.Yellow("Usage: /sec ssl <domain>\nExample: /sec ssl github.com\n")
			return
		}
		color.Cyan("\n🔒 Inspecting SSL/TLS Certificate for %s...\n\n", target)
		res, err := security.InspectSSLCertificate(target)
		if err != nil {
			color.Red("❌ Inspection failed: %v\n", err)
			return
		}
		validTag := color.GreenString("VALID")
		if !res.Valid {
			validTag = color.RedString("INVALID / WARNING")
		}
		color.White("  Status:          %s\n", validTag)
		color.White("  Subject:         %s\n", res.Subject)
		color.White("  Issuer:          %s\n", res.Issuer)
		color.White("  Protocol:        %s\n", res.TLSVersion)
		color.White("  Cipher:          %s\n", res.CipherSuite)
		color.White("  Validity:        %s to %s (%d days remaining)\n", res.NotBefore.Format("2006-01-02"), res.NotAfter.Format("2006-01-02"), res.DaysRemaining)
		if len(res.Warnings) > 0 {
			color.White("\n  ⚠️  Warnings:\n")
			for _, w := range res.Warnings {
				color.HiRed("     • %s\n", w)
			}
		}
		fmt.Println()

	case "ports":
		if len(parts) < 3 {
			target = "localhost"
		}
		color.Cyan("\n🔍 Auditing Common Service Ports for %s...\n\n", target)
		ports := security.AuditPortConnectivity(target, nil)
		openCount := 0
		for _, p := range ports {
			if p.IsOpen {
				openCount++
				color.HiGreen("  🟢 Port %-5d [%-24s] : OPEN  %s\n", p.Port, p.Service, p.Banner)
			}
		}
		if openCount == 0 {
			color.White("  🔒 No standard exposed ports detected.\n")
		}
		fmt.Println()

	case "sast":
		color.Cyan("\n🔍 Running SAST Vulnerability Code Scan on '%s'...\n\n", target)
		res, err := security.ScanCodebase(target)
		if err != nil {
			color.Red("❌ SAST Scan failed: %v\n", err)
			return
		}
		color.White("  Total Files Scanned: %d\n", res.TotalFiles)
		if len(res.Findings) == 0 {
			color.Green("  ✅ Clean! No leaked secrets or high-severity vulnerabilities found.\n")
		} else {
			color.Yellow("  ⚠️  Identified %d findings (Critical: %d, High: %d, Medium: %d):\n\n",
				len(res.Findings), res.CriticalCount, res.HighCount, res.MediumCount)
			for i, f := range res.Findings {
				sevColor := color.HiYellowString
				if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
					sevColor = color.HiRedString
				}
				color.White("  [%s] %s:%d\n    Issue: %s\n    Code:  %s\n    Fix:   %s\n\n",
					sevColor(f.Severity), f.FilePath, f.LineNumber, f.Description, color.HiBlackString(f.Snippet), color.CyanString(f.Remediation))
				if i >= 15 {
					color.HiBlack("  ... and %d more findings\n", len(res.Findings)-15)
					break
				}
			}
		}

	case "audit":
		fmt.Print(security.RunFullAudit(ctx, target))

	default:
		color.Yellow("Unknown security command: %s (Options: headers, ssl, ports, sast, audit)\n", sub)
	}
}
