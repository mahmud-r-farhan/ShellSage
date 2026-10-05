package tui

import (
	"bufio"
	"os"
	"strings"

	"github.com/fatih/color"
	"shellsage/internal/config"
	"shellsage/internal/provider"
)

// RunConfigWizard guides the user through setting up providers, keys, and defaults
func RunConfigWizard(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	color.Cyan("\n⚙️  ━━━━━━━━ ShellSage Configuration & Setup Wizard ━━━━━━━━\n\n")

	// Step 1: Select Active Provider
	selectedProvider := SelectProviderMenu(cfg.ActiveProvider)
	cfg.ActiveProvider = selectedProvider

	pCfg := cfg.Providers[selectedProvider]

	// Step 2: Configure API Key for Active Provider (skipped for local providers)
	desc, _ := provider.ByID(string(selectedProvider))
	if desc.RequiresKey {
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

	// Optional: configure API keys for other providers (multi-provider failover
	// is a core v4 real-world workflow: /provider switch instantly without re-setup).
	for {
		color.Cyan("Configure another provider's key? [y/N]: ")
		line, _ := reader.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			break
		}
		other := SelectProviderMenu(cfg.ActiveProvider)
		pOther := cfg.Providers[other]
		color.White("\n🔑 API Key for %s [Current: %s]:\n", other, config.MaskKey(pOther.APIKey))
		color.Cyan("Enter new API Key (or press Enter to keep current): ")
		keyInput, _ := reader.ReadString('\n')
		keyInput = strings.TrimSpace(keyInput)
		if keyInput != "" {
			pOther.APIKey = keyInput
			cfg.Providers[other] = pOther
		}
		color.White("\n🤖 Default Model for %s [Current: %s] (Enter keeps): ", other, pOther.Model)
		modelInput, _ := reader.ReadString('\n')
		if m := strings.TrimSpace(modelInput); m != "" {
			pOther.Model = m
			cfg.Providers[other] = pOther
		}
		if err := cfg.Save(); err != nil {
			color.Red("❌ Failed to save configuration: %v\n", err)
			return err
		}
		color.Green("✅ Saved configuration for %s.\n", other)
	}
	return nil
}
