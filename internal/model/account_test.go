package model_test

import (
	"encoding/json"
	"testing"

	"Astramail/internal/model"
)

func TestResolvedIMAPHostAndPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email        string
		customHost   string
		customPort   uint16
		expectedHost string
		expectedPort uint16
	}{
		{
			email:        "test@outlook.com",
			expectedHost: "outlook.office365.com",
			expectedPort: 993,
		},
		{
			email:        "user@gmail.com",
			expectedHost: "imap.gmail.com",
			expectedPort: 993,
		},
		{
			email:        "user@qq.com",
			expectedHost: "imap.qq.com",
			expectedPort: 993,
		},
		{
			email:        "user@163.com",
			expectedHost: "imap.163.com",
			expectedPort: 993,
		},
		{
			email:        "custom@mycompany.org",
			expectedHost: "imap.mycompany.org",
			expectedPort: 993,
		},
		{
			email:        "custom@mycompany.org",
			customHost:   "mail.custom.org",
			customPort:   1993,
			expectedHost: "mail.custom.org",
			expectedPort: 1993,
		},
	}

	for _, tt := range tests {
		creds := model.AccountCredentials{
			Email:    tt.email,
			IMAPHost: tt.customHost,
			IMAPPort: tt.customPort,
		}

		if gotHost := creds.ResolvedIMAPHost(); gotHost != tt.expectedHost {
			t.Errorf("email %s: expected host %s, got %s", tt.email, tt.expectedHost, gotHost)
		}

		if gotPort := creds.ResolvedIMAPPort(); gotPort != tt.expectedPort {
			t.Errorf("email %s: expected port %d, got %d", tt.email, tt.expectedPort, gotPort)
		}
	}
}

func TestAccountCredentialsUnmarshal(t *testing.T) {
	t.Parallel()

	snakeJSON := []byte(`{
		"email": "user@example.com",
		"client_id": "cid123",
		"refresh_token": "token456",
		"imap_host": "imap.custom.org",
		"imap_port": 1993,
		"protocol": "both"
	}`)

	var creds1 model.AccountCredentials

	err := json.Unmarshal(snakeJSON, &creds1)
	if err != nil {
		t.Fatalf("unmarshal snake failed: %v", err)
	}

	if creds1.ClientID != "cid123" || creds1.RefreshToken != "token456" {
		t.Errorf("expected cid123 and token456, got %s and %s", creds1.ClientID, creds1.RefreshToken)
	}

	camelJSON := []byte(`{
		"email": "user@example.com",
		"clientId": "cidABC",
		"refreshToken": "tokenXYZ",
		"imapHost": "imap.custom.org",
		"imapPort": 1993,
		"protocol": "both"
	}`)

	var creds2 model.AccountCredentials

	err = json.Unmarshal(camelJSON, &creds2)
	if err != nil {
		t.Fatalf("unmarshal camel failed: %v", err)
	}

	if creds2.ClientID != "cidABC" || creds2.RefreshToken != "tokenXYZ" {
		t.Errorf("expected cidABC and tokenXYZ, got %s and %s", creds2.ClientID, creds2.RefreshToken)
	}
}
