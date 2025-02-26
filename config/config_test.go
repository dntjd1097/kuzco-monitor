package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 테스트용 임시 설정 파일 생성
	testConfig := `
accounts:
  - name: test_account
    kuzco:
      email: test@example.com
      password: test_password
    vastai:
      email: test@example.com
      token: test_token
`
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// 설정 파일 로드 테스트
	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 설정값 검증
	if len(cfg.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(cfg.Accounts))
	}

	account := cfg.Accounts[0]
	if account.Name != "test_account" {
		t.Errorf("Expected account name 'test_account', got '%s'", account.Name)
	}
	if account.Kuzco.Email != "test@example.com" {
		t.Errorf("Expected Kuzco email 'test@example.com', got '%s'", account.Kuzco.Email)
	}
	if account.Vastai.Token != "test_token" {
		t.Errorf("Expected Vastai token 'test_token', got '%s'", account.Vastai.Token)
	}
}
