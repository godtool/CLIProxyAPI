package codeximport

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestConvertFlattenstokensAndBuildsFileName(t *testing.T) {
	t.Helper()

	idToken := testJWT(t, map[string]any{
		"email": "dev@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_team_123456",
			"chatgpt_plan_type":  "team",
		},
	})

	source := []byte(`{
  "OPENAI_API_KEY": null,
  "auth_mode": "chatgpt",
  "last_refresh": "2026-03-24T09:30:00Z",
  "tokens": {
    "access_token": "access-123",
    "refresh_token": "refresh-456",
    "id_token": "` + idToken + `",
    "account_id": "acct_top_level_fallback"
  }
}`)

	result, err := Convert(source, time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result.FileName == "" {
		t.Fatal("Convert() returned empty filename")
	}
	if got, want := result.Metadata["type"], "codex"; got != want {
		t.Fatalf("type = %v, want %v", got, want)
	}
	if got, want := result.Metadata["email"], "dev@example.com"; got != want {
		t.Fatalf("email = %v, want %v", got, want)
	}
	if got, want := result.Metadata["account_id"], "acct_team_123456"; got != want {
		t.Fatalf("account_id = %v, want %v", got, want)
	}
	if got, want := result.Metadata["plan_type"], "team"; got != want {
		t.Fatalf("plan_type = %v, want %v", got, want)
	}
	if got, want := result.Metadata["access_token"], "access-123"; got != want {
		t.Fatalf("access_token = %v, want %v", got, want)
	}
	if got, want := result.Metadata["refresh_token"], "refresh-456"; got != want {
		t.Fatalf("refresh_token = %v, want %v", got, want)
	}
	if _, ok := result.Metadata["expired"].(string); !ok {
		t.Fatalf("expired missing or not a string: %#v", result.Metadata["expired"])
	}
	if result.FileName == "codex-dev@example.com-team.json" {
		t.Fatal("team filename should include hashed account prefix")
	}
	if len(result.FileName) == 0 || result.FileName[len(result.FileName)-5:] != ".json" {
		t.Fatalf("unexpected filename %q", result.FileName)
	}
}

func TestConvertPreservesExplicitExpiryWhenPresent(t *testing.T) {
	idToken := testJWT(t, map[string]any{
		"email": "solo@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_solo_1",
			"chatgpt_plan_type":  "plus",
		},
	})

	source := map[string]any{
		"last_refresh": "2026-03-24T09:30:00Z",
		"tokens": map[string]any{
			"access_token":  "access-abc",
			"refresh_token": "refresh-def",
			"id_token":      idToken,
			"account_id":    "acct_solo_1",
			"expired":       "2026-04-01T00:00:00Z",
		},
	}

	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	result, err := Convert(raw, time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if got, want := result.Metadata["expired"], "2026-04-01T00:00:00Z"; got != want {
		t.Fatalf("expired = %v, want %v", got, want)
	}
	if got, want := result.FileName, "codex-solo@example.com-plus.json"; got != want {
		t.Fatalf("filename = %q, want %q", got, want)
	}
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "."
}
