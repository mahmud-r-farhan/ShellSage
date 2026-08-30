package tui

import (
	"bufio"
	"os"
	"strings"

	"github.com/fatih/color"
	"shellsage/internal/config"
)

// RunConfigWizard guides the user through setting up providers, keys, and defaults
func RunConfigWizard(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan("\n⚙️  ━━━━━━━━ ShellSage Configuration & Setup Wizard ━━━━━━━━\n\n")

	// Step 1: Select Active Provider
	selectedProvider := SelectProviderMenu(cfg.ActiveProvider)
	cfg.ActiveProvider = selectedProvider

	pCfg := cfg.Providers[selectedProvider]

	// Step 2: Configure API Key for Active Provider (skip for Ollama if not needed)
	if selectedProvider != config.ProviderOllama {
		maskedKey := "(Not configured)"
		if len(pCfg.APIKey) > 8 {
			maskedKey = pCfg.APIKey[:4] + "..." + pCfg.APIKey[len(pCfg.APIKey)-4:]
		}

		color.White("\n🔑 API Key for %s [Current: %s]:\n", selectedProvider, maskedKey)
		color.Cyan("Enter new API Key (or press Enter to keep current): ")
		keyInput, _ := reader.ReadString('\n')
		keyInput = strings.TrimSpace(keyInput)
		if keyInput != "" {
			pCfg.APIKey = keyInput
		}
	}

	// Step 3: Base URL
	color.White("\n🌐 Base Endpoint URL [Current: %s]:\n", pCfg.BaseURL)
	color.Cyan("Enter Base URL (or press Enter to keep default): ")
	urlInput, _ := reader.ReadString('\n')
	urlInput = strings.TrimSpace(urlInput)
	if urlInput != "" {
		pCfg.BaseURL = urlInput
	}

	// Step 4: Default Model
	color.White("\n🤖 Default Model [Current: %s]:\n", pCfg.Model)
	color.Cyan("Enter Model ID (or press Enter to keep current): ")
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)
	if modelInput != "" {
		pCfg.Model = modelInput
	}

	cfg.Providers[selectedProvider] = pCfg

	// Step 5: Streaming Toggle
	streamStr := "Enabled (Yes)"
	if !cfg.StreamResponses {
		streamStr = "Disabled (No)"
	}
	color.White("\n⚡ Real-time Token Streaming [Current: %s]:\n", streamStr)
	color.Cyan("Enable streaming? (y/n or press Enter to keep): ")
	streamInput, _ := reader.ReadString('\n')
	streamInput = strings.ToLower(strings.TrimSpace(streamInput))
	if streamInput == "y" || streamInput == "yes" {
		cfg.StreamResponses = true
	} else if streamInput == "n" || streamInput == "no" {
		cfg.StreamResponses = false
	}

	// Step 6: Save Configuration
	if err := cfg.Save(); err != nil {
		color.Red("❌ Failed to save configuration to %s: %v\n", cfg.ConfigPath, err)
		return err
	}

	color.Green("\n✅ Configuration successfully saved to %s!\n\n", cfg.ConfigPath)
	return nil
}
