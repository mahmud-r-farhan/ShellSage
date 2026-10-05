package branch

import (
	"testing"

	"shellsage/internal/provider"
)

func TestImagesRoundTrip(t *testing.T) {
	tree := NewConversationTree("general")
	tree.AddMessageWithParts(provider.Message{Role: "user", Content: "look", Images: []string{"data:image/png;base64,AAA"}}, "m", provider.TokenUsage{})
	msgs := tree.GetActiveMessages()
	if len(msgs) != 1 || len(msgs[0].Images) != 1 {
		t.Fatalf("images must survive GetActiveMessages: %+v", msgs)
	}

	data, err := tree.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := FromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.GetActiveMessages()[0]
	if got.Images == nil || got.Images[0] != "data:image/png;base64,AAA" {
		t.Errorf("images must persist through JSON: %+v", got)
	}
}
