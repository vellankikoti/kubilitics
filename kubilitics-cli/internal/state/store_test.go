package state

import "testing"

func TestMarkContextSwitched(t *testing.T) {
	s := &Store{RecentContexts: []string{"prod", "staging"}}
	s.MarkContextSwitched("prod", "dev")
	if s.LastContext != "prod" {
		t.Fatalf("expected LastContext=prod, got %q", s.LastContext)
	}
	if len(s.RecentContexts) == 0 || s.RecentContexts[0] != "dev" {
		t.Fatalf("expected recent contexts to start with dev, got %+v", s.RecentContexts)
	}
}

func TestFavoritesAddRemove(t *testing.T) {
	s := &Store{}
	s.AddFavorite("prod")
	s.AddFavorite("prod")
	if len(s.Favorites) != 1 {
		t.Fatalf("expected deduplicated favorites, got %v", s.Favorites)
	}
	s.RemoveFavorite("prod")
	if len(s.Favorites) != 0 {
		t.Fatalf("expected favorite removed, got %v", s.Favorites)
	}
}

func TestContextGroupsSetAddRemoveAndActive(t *testing.T) {
	s := &Store{}
	s.SetContextGroup("prod", []string{"ctx-a", "ctx-b", "ctx-a"})
	if got := s.ContextGroups["prod"]; len(got) != 2 || got[0] != "ctx-a" || got[1] != "ctx-b" {
		t.Fatalf("unexpected group members: %v", got)
	}

	s.AddContextGroupMembers("prod", []string{"ctx-c", "ctx-b"})
	if got := s.ContextGroups["prod"]; len(got) != 3 {
		t.Fatalf("expected deduped append to size 3, got %v", got)
	}

	s.SetActiveContextGroup("prod")
	if s.ActiveContextGroup != "prod" {
		t.Fatalf("expected active group prod, got %q", s.ActiveContextGroup)
	}

	s.RemoveContextGroupMembers("prod", []string{"ctx-b"})
	if got := s.ContextGroups["prod"]; len(got) != 2 || got[0] != "ctx-a" || got[1] != "ctx-c" {
		t.Fatalf("unexpected members after remove: %v", got)
	}

	s.RemoveContextGroupMembers("prod", nil)
	if _, ok := s.ContextGroups["prod"]; ok {
		t.Fatalf("expected group deleted")
	}
	if s.ActiveContextGroup != "" {
		t.Fatalf("expected active group cleared, got %q", s.ActiveContextGroup)
	}
}

func TestMarkNamespaceSwitched(t *testing.T) {
	s := &Store{RecentNamespaces: []string{"default", "kube-system"}}
	s.MarkNamespaceSwitched("default", "dev")
	if s.LastNamespace != "default" {
		t.Fatalf("expected LastNamespace=default, got %q", s.LastNamespace)
	}
	if len(s.RecentNamespaces) == 0 || s.RecentNamespaces[0] != "dev" {
		t.Fatalf("expected recent namespaces to start with dev, got %+v", s.RecentNamespaces)
	}
}
