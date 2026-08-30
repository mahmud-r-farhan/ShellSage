package persona

import (
	"testing"
)

func TestPersonaManager(t *testing.T) {
	tempDir := t.TempDir()
	mgr := &PersonaManager{customDir: tempDir}

	// Test built-in personas
	dev, ok := mgr.Get("developer")
	if !ok || dev.ID != "developer" {
		t.Errorf("failed to retrieve built-in developer persona")
	}

	// Test custom persona creation
	custom := Persona{
		ID:          "crypto-expert",
		Name:        "Crypto & Web3 Specialist",
		Description: "Expert in blockchain and smart contracts",
		Prompt:      "You are a blockchain security and smart contract expert.",
	}

	err := mgr.SaveCustom(custom)
	if err != nil {
		t.Fatalf("failed to save custom persona: %v", err)
	}

	retrieved, ok := mgr.Get("crypto-expert")
	if !ok || !retrieved.IsCustom {
		t.Errorf("failed to retrieve custom persona: %+v", retrieved)
	}

	// Test deletion
	err = mgr.DeleteCustom("crypto-expert")
	if err != nil {
		t.Fatalf("failed to delete custom persona: %v", err)
	}

	_, ok = mgr.Get("crypto-expert")
	if ok {
		t.Errorf("custom persona should have been deleted")
	}
}
