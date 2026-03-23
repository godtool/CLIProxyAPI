package codeximport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	codexauth "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codex"
)

const fallbackTokenLifetime = 24 * time.Hour

type sourceAuthFile struct {
	OpenAIAPIKey string           `json:"OPENAI_API_KEY"`
	AuthMode     string           `json:"auth_mode"`
	LastRefresh  string           `json:"last_refresh"`
	Tokens       sourceAuthTokens `json:"tokens"`
}

type sourceAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	AccountID    string `json:"account_id"`
	Expired      string `json:"expired"`
	Expire       string `json:"expire"`
	ExpiresAt    string `json:"expires_at"`
	Expiry       string `json:"expiry"`
}

type Result struct {
	FileName string
	Metadata map[string]any
}

func Convert(source []byte, now time.Time) (*Result, error) {
	if len(source) == 0 {
		return nil, fmt.Errorf("codex import: source auth json is empty")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var input sourceAuthFile
	if err := json.Unmarshal(source, &input); err != nil {
		return nil, fmt.Errorf("codex import: decode source file: %w", err)
	}

	accessToken := strings.TrimSpace(input.Tokens.AccessToken)
	refreshToken := strings.TrimSpace(input.Tokens.RefreshToken)
	idToken := strings.TrimSpace(input.Tokens.IDToken)
	accountID := strings.TrimSpace(input.Tokens.AccountID)
	lastRefresh := strings.TrimSpace(input.LastRefresh)
	if accessToken == "" || refreshToken == "" || idToken == "" {
		return nil, fmt.Errorf("codex import: source auth file missing required tokens")
	}
	if lastRefresh == "" {
		lastRefresh = now.UTC().Format(time.RFC3339)
	}

	email := ""
	planType := ""
	if claims, err := codexauth.ParseJWTToken(idToken); err == nil && claims != nil {
		email = strings.TrimSpace(claims.GetUserEmail())
		planType = strings.TrimSpace(claims.CodexAuthInfo.ChatgptPlanType)
		if accountFromJWT := strings.TrimSpace(claims.GetAccountID()); accountFromJWT != "" {
			accountID = accountFromJWT
		}
	}

	if email == "" {
		return nil, fmt.Errorf("codex import: unable to derive email from id_token")
	}
	if accountID == "" {
		return nil, fmt.Errorf("codex import: source auth file missing account_id")
	}

	expired := firstNonEmpty(
		strings.TrimSpace(input.Tokens.Expired),
		strings.TrimSpace(input.Tokens.Expire),
		strings.TrimSpace(input.Tokens.ExpiresAt),
		strings.TrimSpace(input.Tokens.Expiry),
	)
	if expired == "" {
		expired = now.UTC().Add(fallbackTokenLifetime).Format(time.RFC3339)
	}

	hashAccountID := ""
	if strings.EqualFold(planType, "team") {
		digest := sha256.Sum256([]byte(accountID))
		hashAccountID = hex.EncodeToString(digest[:])[:8]
	}

	fileName := codexauth.CredentialFileName(email, planType, hashAccountID, true)
	metadata := map[string]any{
		"type":          "codex",
		"email":         email,
		"account_id":    accountID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"id_token":      idToken,
		"expired":       expired,
		"last_refresh":  lastRefresh,
	}
	if trimmedPlan := strings.TrimSpace(planType); trimmedPlan != "" {
		metadata["plan_type"] = trimmedPlan
	}
	if apiKey := strings.TrimSpace(input.OpenAIAPIKey); apiKey != "" {
		metadata["api_key"] = apiKey
	}
	if authMode := strings.TrimSpace(input.AuthMode); authMode != "" {
		metadata["auth_mode"] = authMode
	}

	return &Result{
		FileName: fileName,
		Metadata: metadata,
	}, nil
}

func MarshalOutput(result *Result) ([]byte, error) {
	if result == nil {
		return nil, fmt.Errorf("codex import: result is nil")
	}
	return json.MarshalIndent(result.Metadata, "", "  ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
