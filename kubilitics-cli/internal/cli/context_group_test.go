package cli

import "testing"

func TestDecodeContextGroupPayload(t *testing.T) {
	t.Run("with metadata envelope", func(t *testing.T) {
		active, groups, err := decodeContextGroupPayload([]byte(`{"activeGroup":"prod","groups":{"prod":["ctx-a","ctx-b"]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active != "prod" {
			t.Fatalf("expected active group prod, got %q", active)
		}
		if len(groups["prod"]) != 2 {
			t.Fatalf("expected 2 members, got %v", groups["prod"])
		}
	})

	t.Run("plain map", func(t *testing.T) {
		active, groups, err := decodeContextGroupPayload([]byte(`{"prod":["ctx-a"],"dev":["ctx-b"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active != "" {
			t.Fatalf("expected empty active group, got %q", active)
		}
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(groups))
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		_, _, err := decodeContextGroupPayload([]byte(`{"groups":"bad"}`))
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}
