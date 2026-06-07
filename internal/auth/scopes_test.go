package auth

import "testing"

func TestIsValidScope_KnownScopes(t *testing.T) {
	for _, scope := range AllScopes {
		if !IsValidScope(scope) {
			t.Errorf("IsValidScope(%q) = false, want true (scope is in AllScopes)", scope)
		}
	}
}

func TestIsValidScope_UnknownScopes(t *testing.T) {
	invalid := []string{
		"",
		"unknown",
		" Admin", // leading space
		"admin ", // trailing space
		"deploy:readonly",
		"read write", // multiple scopes in one string
		"administer",
		"super",
		"role:admin",
	}
	for _, scope := range invalid {
		if IsValidScope(scope) {
			t.Errorf("IsValidScope(%q) = true, want false", scope)
		}
	}
}

func TestValidateScopes_FiltersInvalid(t *testing.T) {
	in := []string{"read", "bogus", "write", "", "admin:full", ScopeDeploy}
	got := ValidateScopes(in)
	want := []string{"read", "write", ScopeDeploy}

	if len(got) != len(want) {
		t.Fatalf("ValidateScopes length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i, s := range want {
		if got[i] != s {
			t.Errorf("ValidateScopes[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestValidateScopes_AllValid(t *testing.T) {
	got := ValidateScopes(AllScopes)
	if len(got) != len(AllScopes) {
		t.Fatalf("ValidateScopes(AllScopes) length = %d, want %d", len(got), len(AllScopes))
	}
}

func TestValidateScopes_AllInvalid(t *testing.T) {
	in := []string{"", "foo", "bar"}
	got := ValidateScopes(in)
	if len(got) != 0 {
		t.Errorf("ValidateScopes(all invalid) = %v, want empty slice", got)
	}
}

func TestValidateScopes_PreservesDuplicates(t *testing.T) {
	// The function does not deduplicate; verify documented behavior.
	in := []string{"read", "read", "write", "read"}
	got := ValidateScopes(in)
	want := []string{"read", "read", "write", "read"}
	if len(got) != len(want) {
		t.Fatalf("ValidateScopes(duplicates) length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidateScopes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateScopes_EmptyInput(t *testing.T) {
	got := ValidateScopes(nil)
	if len(got) != 0 {
		t.Errorf("ValidateScopes(nil) = %v, want empty", got)
	}
}

func TestAllScopes_ContainsExpectedCoreScopes(t *testing.T) {
	required := []string{
		ScopeRead,
		ScopeWrite,
		ScopeDelete,
		ScopeDeploy,
		ScopeAdmin,
	}
	for _, s := range required {
		found := false
		for _, a := range AllScopes {
			if a == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllScopes missing required core scope %q", s)
		}
	}
}

func TestScopeDescriptions_AllScopesDocumented(t *testing.T) {
	for _, scope := range AllScopes {
		desc, ok := ScopeDescriptions[scope]
		if !ok {
			t.Errorf("scope %q has no description in ScopeDescriptions", scope)
			continue
		}
		if desc == "" {
			t.Errorf("scope %q has empty description", scope)
		}
	}
}

func TestScopeDescriptions_NoOrphanedEntries(t *testing.T) {
	// Every entry in ScopeDescriptions should be a valid scope, otherwise
	// a future rename of a scope constant would silently leave a stale
	// description behind.
	for scope := range ScopeDescriptions {
		if !IsValidScope(scope) {
			t.Errorf("ScopeDescriptions contains entry for unknown scope %q", scope)
		}
	}
}

func TestAllScopes_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(AllScopes))
	for _, s := range AllScopes {
		if seen[s] {
			t.Errorf("duplicate scope in AllScopes: %q", s)
		}
		seen[s] = true
	}
}
