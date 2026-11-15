package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name                 string
		addr                 string
		databaseURL          string
		wantAddr             string
		wantErr              bool
		knowledgeBaseDir     string
		wantKnowledgeBaseDir string
	}{
		{
			name:                 "default address",
			databaseURL:          "postgres://localhost/buddy",
			wantAddr:             ":8080",
			wantKnowledgeBaseDir: "../knowledge-base",
		},
		{
			name:                 "explicit address",
			addr:                 "127.0.0.1:9090",
			databaseURL:          "postgres://localhost/buddy",
			wantAddr:             "127.0.0.1:9090",
			knowledgeBaseDir:     "/data/wiki",
			wantKnowledgeBaseDir: "/data/wiki",
		},
		{
			name:    "database URL required",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDR", tt.addr)
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("KNOWLEDGE_BASE_DIR", tt.knowledgeBaseDir)
			for key, value := range map[string]string{
				"ADDR": tt.addr, "DATABASE_URL": tt.databaseURL, "KNOWLEDGE_BASE_DIR": tt.knowledgeBaseDir,
			} {
				if value == "" {
					if err := os.Unsetenv(key); err != nil {
						t.Fatal(err)
					}
				}
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && (cfg.Addr != tt.wantAddr || cfg.DatabaseURL != tt.databaseURL) {
				t.Fatalf("Load() = %+v, want Addr %q, DatabaseURL %q", cfg, tt.wantAddr, tt.databaseURL)
			}
			if !tt.wantErr && cfg.KnowledgeBaseDir != tt.wantKnowledgeBaseDir {
				t.Fatalf("KnowledgeBaseDir = %q, want %q", cfg.KnowledgeBaseDir, tt.wantKnowledgeBaseDir)
			}
		})
	}
}
