package history

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"shellsage/internal/branch"
)

func TestSaveLoadRenameDeleteInHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLSAGE_HOME", home)

	tree := branch.NewConversationTree("general")
	tree.Title = "Session Under Test"
	tree.AddMessage("user", "alpha prompt", "m", branch.TreeNode{}.Usage)
	tree.AddMessage("assistant", "beta answer", "m", branch.TreeNode{}.Usage)

	path, err := SaveTree(tree, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Join(home, "conversations") {
		t.Errorf("sessions should live under SHELLSAGE_HOME/conversations, got %s", path)
	}

	fi, _ := os.Stat(path)
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0600 { // NTFS has no POSIX mode bits
		t.Errorf("session files should be 0600 (may contain proprietary code), got %04o", fi.Mode().Perm())
	}

	list := ListSavedConversations()
	if len(list) != 1 || list[0].Filename != filepath.Base(path) {
		t.Fatalf("list mismatch: %+v", list)
	}

	loaded, err := LoadTree(filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Session Under Test" {
		t.Errorf("title lost: %q", loaded.Title)
	}

	newPath, err := RenameTree(filepath.Base(path), "renamed.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("rename target missing: %v", err)
	}

	if _, err := DeleteTree("renamed.json"); err != nil {
		t.Errorf("delete failed: %v", err)
	}
	if len(ListSavedConversations()) != 0 {
		t.Error("list should be empty after delete")
	}
}

func TestLegacyConversationsDirMerges(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLSAGE_HOME", filepath.Join(dir, "home"))
	t.Chdir(dir) // legacy ./conversations lives in CWD

	legacyDir := filepath.Join(dir, "conversations")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	tree := branch.NewConversationTree("general")
	tree.AddMessage("user", "old style", "m", branch.TreeNode{}.Usage)
	data, err := tree.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "legacy.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	list := ListSavedConversations()
	found := false
	for _, c := range list {
		if c.Filename == "legacy.json" {
			found = true
		}
	}
	if !found {
		t.Errorf("pre-v4 ./conversations sessions must still be listed: %+v", list)
	}
}
