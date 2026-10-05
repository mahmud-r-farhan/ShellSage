package provider

import (
	"sort"
	"strings"
)

// Descriptor is the single source of truth for a supported LLM provider.
//
// Adding a new provider means appending ONE entry here: config defaults,
// environment-variable overrides (<ENV_PREFIX>_API_KEY/_MODEL/_BASE_URL),
// the interactive menus, `shellsage provider list` and the factory all derive
// from this table.
type Descriptor struct {
	// ID is the canonical identifier used in config files, CLI flags and URLs.
	ID string
	// DisplayName is the pretty name shown in menus and error messages.
	DisplayName string
	// Tagline is a one-line description for interactive menus.
	Tagline string
	// Format selects the wire protocol: "openai" (OpenAI-compatible),
	// "anthropic" (Claude Messages API) or "ollama" (local Ollama server).
	Format string
	// EnvPrefix drives generic env overrides: PREFIX_API_KEY, PREFIX_MODEL, PREFIX_BASE_URL.
	EnvPrefix string
	// KeyEnvs are additional/alternative environment variables checked for the API key
	// (e.g. GITHUB_TOKEN, GH_TOKEN). The first non-empty value wins, after PREFIX_API_KEY.
	KeyEnvs []string
	// BaseURLEnv is an optional alternative env var for the endpoint (e.g. AZURE_OPENAI_ENDPOINT).
	BaseURLEnv string
	// DefaultBaseURL is used when the user configured nothing.
	DefaultBaseURL string
	// DefaultModel is used when the user configured nothing.
	DefaultModel string
	// Models is the curated quick-pick list offered in menus.
	Models []string
	// ExtraHeaders are attached to every API request (e.g. OpenRouter attribution).
	ExtraHeaders map[string]string
	// AuthStyle: "" => Authorization: Bearer <key>; "api-key" => `api-key: <key>` header (Azure).
	AuthStyle string
	// RequiresKey is false for local servers (Ollama) that work without auth.
	RequiresKey bool
	// DocsURL points at the provider's API docs / key console.
	DocsURL string
}

// Format identifiers for Descriptor.Format.
const (
	FormatOpenAI    = "openai"
	FormatAnthropic = "anthropic"
	FormatOllama    = "ollama"
)

// registry lists every supported provider. Order matters: it drives menu
// ordering (most-used first) and `provider list` output.
var registry = []Descriptor{
	{
		ID:             "openrouter",
		DisplayName:    "OpenRouter",
		Tagline:        "Unified hub for 300+ models (free & paid)",
		Format:         FormatOpenAI,
		EnvPrefix:      "OPENROUTER",
		DefaultBaseURL: "https://openrouter.ai/api/v1",
		DefaultModel:   "openrouter/auto",
		Models: []string{
			"openrouter/auto",
			"openrouter/free",
			"anthropic/claude-3.5-sonnet",
			"openai/gpt-4o-mini",
			"google/gemini-2.0-flash-001",
			"deepseek/deepseek-chat-v3-0324",
			"meta-llama/llama-3.3-70b-instruct",
			"qwen/qwen-2.5-coder-32b-instruct",
		},
		ExtraHeaders: map[string]string{
			"HTTP-Referer": "https://github.com/mahmud-r-farhan/ShellSage",
			"X-Title":      "ShellSage CLI",
		},
		RequiresKey: true,
		DocsURL:     "https://openrouter.ai/keys",
	},
	{
		ID:             "openai",
		DisplayName:    "OpenAI",
		Tagline:        "GPT-4.1, GPT-4o, o-series reasoning models",
		Format:         FormatOpenAI,
		EnvPrefix:      "OPENAI",
		DefaultBaseURL: "https://api.openai.com/v1",
		DefaultModel:   "gpt-4o-mini",
		Models: []string{
			"gpt-4.1",
			"gpt-4.1-mini",
			"gpt-4o",
			"gpt-4o-mini",
			"o3",
			"o4-mini",
		},
		RequiresKey: true,
		DocsURL:     "https://platform.openai.com/api-keys",
	},
	{
		ID:             "anthropic",
		DisplayName:    "Anthropic Claude",
		Tagline:        "Claude 4 Sonnet/Opus & Claude 3.5 family",
		Format:         FormatAnthropic,
		EnvPrefix:      "ANTHROPIC",
		DefaultBaseURL: "https://api.anthropic.com/v1",
		DefaultModel:   "claude-3-5-haiku-latest",
		Models: []string{
			"claude-sonnet-4-20250514",
			"claude-opus-4-20250514",
			"claude-3-7-sonnet-latest",
			"claude-3-5-sonnet-latest",
			"claude-3-5-haiku-latest",
		},
		RequiresKey: true,
		DocsURL:     "https://console.anthropic.com/settings/keys",
	},
	{
		ID:             "gemini",
		DisplayName:    "Google Gemini",
		Tagline:        "Gemini 2.5 Pro/Flash via OpenAI-compatible endpoint",
		Format:         FormatOpenAI,
		EnvPrefix:      "GEMINI",
		KeyEnvs:        []string{"GOOGLE_API_KEY"},
		DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		DefaultModel:   "gemini-2.0-flash",
		Models: []string{
			"gemini-2.5-pro",
			"gemini-2.5-flash",
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-2.5-flash-lite-preview-06-17",
		},
		RequiresKey: true,
		DocsURL:     "https://aistudio.google.com/apikey",
	},
	{
		ID:             "groq",
		DisplayName:    "Groq Cloud",
		Tagline:        "Ultra-fast LPU inference (500+ tok/s)",
		Format:         FormatOpenAI,
		EnvPrefix:      "GROQ",
		DefaultBaseURL: "https://api.groq.com/openai/v1",
		DefaultModel:   "llama-3.3-70b-versatile",
		Models: []string{
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"deepseek-r1-distill-llama-70b",
			"gpt-oss-120b",
			"gpt-oss-20b",
			"meta-llama/llama-4-scout-17b-16e-instruct",
		},
		RequiresKey: true,
		DocsURL:     "https://console.groq.com/keys",
	},
	{
		ID:             "deepseek",
		DisplayName:    "DeepSeek",
		Tagline:        "DeepSeek-V3 & DeepSeek-R1 reasoning",
		Format:         FormatOpenAI,
		EnvPrefix:      "DEEPSEEK",
		DefaultBaseURL: "https://api.deepseek.com/v1",
		DefaultModel:   "deepseek-chat",
		Models: []string{
			"deepseek-chat",
			"deepseek-reasoner",
		},
		RequiresKey: true,
		DocsURL:     "https://platform.deepseek.com/api_keys",
	},
	{
		ID:             "mistral",
		DisplayName:    "Mistral AI",
		Tagline:        "Mistral Large & Codestral (EU frontier models)",
		Format:         FormatOpenAI,
		EnvPrefix:      "MISTRAL",
		DefaultBaseURL: "https://api.mistral.ai/v1",
		DefaultModel:   "mistral-large-latest",
		Models: []string{
			"mistral-large-latest",
			"mistral-small-latest",
			"codestral-latest",
			"open-mistral-nemo",
			"open-codestral-mamba",
		},
		RequiresKey: true,
		DocsURL:     "https://console.mistral.ai/api-keys",
	},
	{
		ID:             "xai",
		DisplayName:    "xAI (Grok)",
		Tagline:        "Grok 4 / Grok 3 with 128k context",
		Format:         FormatOpenAI,
		EnvPrefix:      "XAI",
		DefaultBaseURL: "https://api.x.ai/v1",
		DefaultModel:   "grok-3",
		Models: []string{
			"grok-4",
			"grok-3",
			"grok-3-mini",
			"grok-2-vision-1212",
		},
		RequiresKey: true,
		DocsURL:     "https://console.x.ai",
	},
	{
		ID:             "together",
		DisplayName:    "Together AI",
		Tagline:        "Open-source model hosting (Llama, DeepSeek, Qwen)",
		Format:         FormatOpenAI,
		EnvPrefix:      "TOGETHER",
		KeyEnvs:        []string{"TOGETHERAI_API_KEY"},
		DefaultBaseURL: "https://api.together.xyz/v1",
		DefaultModel:   "deepseek-ai/DeepSeek-V3",
		Models: []string{
			"deepseek-ai/DeepSeek-V3",
			"deepseek-ai/DeepSeek-R1",
			"meta-llama/Llama-3.3-70B-Instruct-Turbo",
			"Qwen/Qwen2.5-Coder-32B-Instruct-Turbo",
			"mistralai/Mixtral-8x7B-Instruct-v0.1",
		},
		RequiresKey: true,
		DocsURL:     "https://api.together.xyz/settings/api-keys",
	},
	{
		ID:             "fireworks",
		DisplayName:    "Fireworks AI",
		Tagline:        "High-throughput inference on frontier open models",
		Format:         FormatOpenAI,
		EnvPrefix:      "FIREWORKS",
		DefaultBaseURL: "https://api.fireworks.ai/inference/v1",
		DefaultModel:   "accounts/fireworks/models/llama-v3p1-70b-instruct",
		Models: []string{
			"accounts/fireworks/models/llama-v3p1-70b-instruct",
			"accounts/fireworks/models/deepseek-v3",
			"accounts/fireworks/models/qwen2p5-coder-32b-instruct",
			"accounts/fireworks/models/firefunction-v2",
		},
		RequiresKey: true,
		DocsURL:     "https://app.fireworks.ai/settings/users/api-keys",
	},
	{
		ID:             "cerebras",
		DisplayName:    "Cerebras Inference",
		Tagline:        "~1000 tok/s wafer-scale inference, generous free tier",
		Format:         FormatOpenAI,
		EnvPrefix:      "CEREBRAS",
		DefaultBaseURL: "https://api.cerebras.ai/v1",
		DefaultModel:   "llama3.1-8b",
		Models: []string{
			"llama3.1-8b",
			"llama3.1-70b",
			"llama-3.3-70b",
			"qwen-3-32b",
			"qwen-3-coder-480b",
		},
		RequiresKey: true,
		DocsURL:     "https://cloud.cerebras.ai",
	},
	{
		ID:             "nvidia",
		DisplayName:    "NVIDIA NIM",
		Tagline:        "Optimized models on NVIDIA GPU cloud (free credits)",
		Format:         FormatOpenAI,
		EnvPrefix:      "NVIDIA",
		DefaultBaseURL: "https://integrate.api.nvidia.com/v1",
		DefaultModel:   "deepseek-ai/deepseek-r1",
		Models: []string{
			"deepseek-ai/deepseek-r1",
			"meta-llama/llama-3.3-70b-instruct",
			"nvidia/llama-3.1-nemotron-70b-instruct",
			"moonshotai/kimi-k2-instruct",
		},
		RequiresKey: true,
		DocsURL:     "https://build.nvidia.com",
	},
	{
		ID:             "github",
		DisplayName:    "GitHub Models",
		Tagline:        "Use your GitHub token — no extra signup (free tier)",
		Format:         FormatOpenAI,
		EnvPrefix:      "GITHUB",
		KeyEnvs:        []string{"GITHUB_TOKEN", "GH_TOKEN"},
		DefaultBaseURL: "https://models.inference.ai.azure.com",
		DefaultModel:   "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4.1",
			"Meta-Llama-3.3-70B-Instruct",
			"mistral-large-2407",
		},
		RequiresKey: true,
		DocsURL:     "https://github.com/marketplace/models",
	},
	{
		ID:             "huggingface",
		DisplayName:    "Hugging Face Inference",
		Tagline:        "Router for thousands of open models",
		Format:         FormatOpenAI,
		EnvPrefix:      "HUGGINGFACE",
		KeyEnvs:        []string{"HF_TOKEN"},
		DefaultBaseURL: "https://router.huggingface.co/v1",
		DefaultModel:   "meta-llama/Llama-3.3-70B-Instruct",
		Models: []string{
			"meta-llama/Llama-3.3-70B-Instruct",
			"deepseek-ai/DeepSeek-V3",
			"Qwen/Qwen2.5-Coder-32B-Instruct",
			"microsoft/Phi-4",
		},
		RequiresKey: true,
		DocsURL:     "https://huggingface.co/settings/tokens",
	},
	{
		ID:             "perplexity",
		DisplayName:    "Perplexity",
		Tagline:        "Sonar models with built-in live web search",
		Format:         FormatOpenAI,
		EnvPrefix:      "PERPLEXITY",
		DefaultBaseURL: "https://api.perplexity.ai",
		DefaultModel:   "sonar",
		Models: []string{
			"sonar",
			"sonar-pro",
			"sonar-reasoning-pro",
		},
		RequiresKey: true,
		DocsURL:     "https://www.perplexity.ai/settings/api",
	},
	{
		ID:             "deepinfra",
		DisplayName:    "DeepInfra",
		Tagline:        "Low-cost serverless open models",
		Format:         FormatOpenAI,
		EnvPrefix:      "DEEPINFRA",
		DefaultBaseURL: "https://api.deepinfra.com/v1/openai",
		DefaultModel:   "deepseek-ai/DeepSeek-V3",
		Models: []string{
			"deepseek-ai/DeepSeek-V3",
			"deepseek-ai/DeepSeek-R1",
			"Qwen/Qwen3-32B",
			"meta-llama/Llama-3.3-70B-Instruct",
		},
		RequiresKey: true,
		DocsURL:     "https://deepinfra.com/dash/api_keys",
	},
	{
		ID:             "cohere",
		DisplayName:    "Cohere",
		Tagline:        "Command A / R-series for enterprise & RAG",
		Format:         FormatOpenAI,
		EnvPrefix:      "COHERE",
		DefaultBaseURL: "https://api.cohere.com/v2",
		DefaultModel:   "command-a-03-2025",
		Models: []string{
			"command-a-03-2025",
			"command-r-08-2024",
			"command-r-plus-08-2024",
		},
		RequiresKey: true,
		DocsURL:     "https://dashboard.cohere.com/api-keys",
	},
	{
		ID:             "azure",
		DisplayName:    "Azure OpenAI",
		Tagline:        "Enterprise deployment: base URL + model = deployment name",
		Format:         FormatOpenAI,
		EnvPrefix:      "AZURE_OPENAI",
		BaseURLEnv:     "AZURE_OPENAI_ENDPOINT",
		DefaultBaseURL: "https://YOUR-RESOURCE.openai.azure.com/openai/v1",
		DefaultModel:   "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4.1",
			"o3-mini",
		},
		AuthStyle:   "api-key",
		RequiresKey: true,
		DocsURL:     "https://learn.microsoft.com/azure/ai-services/openai/",
	},
	{
		ID:             "ollama",
		DisplayName:    "Ollama (Local)",
		Tagline:        "Private offline models on your machine",
		Format:         FormatOllama,
		EnvPrefix:      "OLLAMA",
		DefaultBaseURL: "http://localhost:11434/v1",
		DefaultModel:   "llama3.2:latest",
		Models: []string{
			"llama3.2:latest",
			"llama3.1:latest",
			"qwen2.5-coder:latest",
			"deepseek-r1:8b",
			"gemma3:latest",
		},
		RequiresKey: false,
		DocsURL:     "https://ollama.com/library",
	},
	{
		ID:             "custom",
		DisplayName:    "Custom / Self-Hosted",
		Tagline:        "Any OpenAI-compatible server (vLLM, LM Studio, llama.cpp)",
		Format:         FormatOpenAI,
		EnvPrefix:      "SHELLSAGE_CUSTOM",
		DefaultBaseURL: "http://localhost:8080/v1",
		DefaultModel:   "default-model",
		Models:         nil,
		RequiresKey:    false,
		DocsURL:        "",
	},
}

// All returns every provider descriptor (menu order).
func All() []Descriptor {
	out := make([]Descriptor, len(registry))
	copy(out, registry)
	return out
}

// IDs returns all provider ids in registry order.
func IDs() []string {
	out := make([]string, 0, len(registry))
	for _, d := range registry {
		out = append(out, d.ID)
	}
	return out
}

// ByID looks up a descriptor case-insensitively, with a few aliases.
func ByID(id string) (Descriptor, bool) {
	lower := strings.ToLower(strings.TrimSpace(id))
	// Common aliases keep muscle memory and old configs working.
	switch lower {
	case "claude":
		lower = "anthropic"
	case "google", "bard":
		lower = "gemini"
	case "gh", "github-models", "githubmodels":
		lower = "github"
	case "hf", "hugging-face", "hugging_face":
		lower = "huggingface"
	case "grok":
		lower = "xai"
	case "azure-openai", "aoai":
		lower = "azure"
	case "lmstudio", "lm-studio", "vllm", "llamacpp", "localai":
		lower = "custom"
	}
	for _, d := range registry {
		if d.ID == lower {
			return d, true
		}
	}
	return Descriptor{}, false
}

// NormalizeID canonicalizes a (possibly aliased) provider id.
func NormalizeID(id string) (string, bool) {
	d, ok := ByID(id)
	if !ok {
		return id, false
	}
	return d.ID, true
}

// SortedIDs returns provider ids sorted alphabetically (for `provider list`).
func SortedIDs() []string {
	ids := IDs()
	sort.Strings(ids)
	return ids
}

// Selection is the explicit provider configuration the factory needs.
// Kept dependency-free so `provider` never imports `config` (no import cycles).
type Selection struct {
	ID      string
	APIKey  string
	BaseURL string
	Model   string
}

// New constructs the Provider implementation described by the registry entry
// matching sel.ID (falls back to the descriptor's defaults for empty fields).
func New(sel Selection) (Provider, error) {
	desc, ok := ByID(sel.ID)
	if !ok {
		return nil, &UnknownProviderError{ID: sel.ID}
	}
	baseURL := sel.BaseURL
	if baseURL == "" {
		baseURL = desc.DefaultBaseURL
	}
	model := sel.Model
	if model == "" {
		model = desc.DefaultModel
	}

	switch desc.Format {
	case FormatAnthropic:
		return NewAnthropicProvider(sel.APIKey, baseURL, model), nil
	case FormatOllama:
		return NewOllamaProvider(baseURL, model), nil
	default:
		base := NewBaseProvider(desc.DisplayName, sel.APIKey, baseURL, model, desc.Models, desc.ExtraHeaders)
		base.SetAuthStyle(desc.AuthStyle)
		return base, nil
	}
}

// UnknownProviderError is returned by New for ids not in the registry.
type UnknownProviderError struct{ ID string }

func (e *UnknownProviderError) Error() string {
	return "unknown provider '" + e.ID + "' — run `shellsage provider list` for available providers (" +
		strings.Join(IDs(), ", ") + ")"
}
