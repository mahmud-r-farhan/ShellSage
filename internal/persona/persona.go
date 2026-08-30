package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"shellsage/internal/config"
)

// Persona represents an AI persona with a distinct system prompt and attributes
type Persona struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	IsCustom    bool     `json:"is_custom"`
	Tags        []string `json:"tags,omitempty"`
}

// Built-in personas
var builtInPersonas = map[string]Persona{
	"general": {
		ID:          "general",
		Name:        "General Assistant",
		Description: "Helpful, concise, respectful, and balanced AI assistant",
		Prompt:      "You are ShellSage, a helpful, precise, and concise AI terminal assistant. Provide clear, accurate, and direct answers without unnecessary fluff.",
		Tags:        []string{"general", "default"},
	},
	"developer": {
		ID:          "developer",
		Name:        "Senior Software Engineer",
		Description: "Expert full-stack & systems developer specializing in clean code and design patterns",
		Prompt:      "You are a Senior Principal Software Engineer. Provide idiomatic, clean, robust, and well-structured code. Emphasize best practices, error handling, performance, modularity, and testability. Include clear explanations and comments where appropriate.",
		Tags:        []string{"code", "engineering"},
	},
	"architect": {
		ID:          "architect",
		Name:        "System Architect",
		Description: "Focuses on high-level system design, scalability, microservices, and design trade-offs",
		Prompt:      "You are an Enterprise System Architect. Analyze problems from high-level perspectives: scalability, resilience, fault tolerance, API contract design, data flow, distributed systems, and architectural trade-offs.",
		Tags:        []string{"architecture", "design"},
	},
	"debugger": {
		ID:          "debugger",
		Name:        "Debug & Root-Cause Specialist",
		Description: "Diagnostic expert for stack traces, memory leaks, concurrency bugs, and logic errors",
		Prompt:      "You are a Root-Cause Debugging Expert. Analyze error messages, stack traces, race conditions, and bug reports. Pinpoint the exact failure mechanism, explain why it occurred, and provide the minimal robust fix with regression prevention tips.",
		Tags:        []string{"debug", "troubleshooting"},
	},
	"docgen": {
		ID:          "docgen",
		Name:        "Technical Writer & Doc Creator",
		Description: "Creates crystal-clear documentation, API references, READMEs, and tutorials",
		Prompt:      "You are an expert Technical Documentation Specialist. Write clear, engaging, and comprehensive technical documentation, READMEs, architecture summaries, and API documentation formatted in clean GitHub-Flavored Markdown.",
		Tags:        []string{"docs", "writing"},
	},
	"devops": {
		ID:          "devops",
		Name:        "DevOps & SRE Engineer",
		Description: "Specialist in CI/CD, Docker, Kubernetes, Terraform, Cloud, and Linux systems",
		Prompt:      "You are a Senior DevOps & Site Reliability Engineer. Provide production-ready configurations for Docker, Kubernetes, CI/CD pipelines (GitHub Actions, GitLab CI), Infrastructure as Code (Terraform), and Linux sysadmin troubleshooting.",
		Tags:        []string{"devops", "cloud"},
	},
	"security": {
		ID:          "security",
		Name:        "Security Auditor & Code Reviewer",
		Description: "Identifies vulnerabilities (OWASP), unsafe memory access, auth flaws, and hardening",
		Prompt:      "You are a Cybersecurity Specialist and Senior Code Reviewer. Audit code for security vulnerabilities, injection flaws, memory safety issues, cryptographic misuse, and authorization flaws. Suggest hardened remediation.",
		Tags:        []string{"security", "audit"},
	},
	"planner": {
		ID:          "planner",
		Name:        "Agile Project & Task Planner",
		Description: "Breaks complex projects into actionable milestones, epics, and checklists",
		Prompt:      "You are a Technical Project Planner and Agile Architect. Break down complex product and engineering goals into structured, actionable, dependency-aware milestones, step-by-step tasks, and verification criteria.",
		Tags:        []string{"planning", "management"},
	},
}

// GetPersonasDir returns the directory path for custom personas
func GetPersonasDir() string {
	dir := filepath.Join(config.GetUserConfigDir(), "personas")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// PersonaManager handles loading, registering, and listing personas
type PersonaManager struct {
	customDir string
}

// NewPersonaManager initializes the manager
func NewPersonaManager() *PersonaManager {
	return &PersonaManager{
		customDir: GetPersonasDir(),
	}
}

// GetAll returns a merged list of built-in and custom personas
func (m *PersonaManager) GetAll() map[string]Persona {
	result := make(map[string]Persona)
	for k, v := range builtInPersonas {
		result[k] = v
	}

	// Load local custom personas
	entries, err := os.ReadDir(m.customDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				filePath := filepath.Join(m.customDir, entry.Name())
				if data, err := os.ReadFile(filePath); err == nil {
					var p Persona
					if err := json.Unmarshal(data, &p); err == nil && p.ID != "" {
						p.IsCustom = true
						result[p.ID] = p
					}
				}
			}
		}
	}

	return result
}

// Get finds a persona by ID or name
func (m *PersonaManager) Get(idOrName string) (Persona, bool) {
	all := m.GetAll()
	lower := strings.ToLower(strings.TrimSpace(idOrName))

	// Direct match
	if p, ok := all[lower]; ok {
		return p, true
	}

	// Match by Name or partial ID
	for _, p := range all {
		if strings.ToLower(p.Name) == lower || strings.EqualFold(p.ID, lower) {
			return p, true
		}
	}

	return builtInPersonas["general"], false
}

// SaveCustom saves a custom persona to the user's persona directory
func (m *PersonaManager) SaveCustom(p Persona) error {
	if p.ID == "" {
		p.ID = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))
	}
	p.IsCustom = true

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode persona: %w", err)
	}

	filePath := filepath.Join(m.customDir, p.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save persona to %s: %w", filePath, err)
	}

	return nil
}

// DeleteCustom removes a custom persona file
func (m *PersonaManager) DeleteCustom(id string) error {
	filePath := filepath.Join(m.customDir, id+".json")
	return os.Remove(filePath)
}

// ListSorted returns personas sorted by name
func (m *PersonaManager) ListSorted() []Persona {
	all := m.GetAll()
	list := make([]Persona, 0, len(all))
	for _, p := range all {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].IsCustom != list[j].IsCustom {
			return !list[i].IsCustom // built-in first
		}
		return list[i].Name < list[j].Name
	})

	return list
}
