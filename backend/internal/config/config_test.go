package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		databaseURL string
		wantAddr    string
		wantErr     bool
	}{
		{
			name:        "default address",
			databaseURL: "postgres://localhost/buddy",
			wantAddr:    ":8080",
		},
		{
			name:        "explicit address",
			addr:        "127.0.0.1:9090",
			databaseURL: "postgres://localhost/buddy",
			wantAddr:    "127.0.0.1:9090",
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
			for key, value := range map[string]string{"ADDR": tt.addr, "DATABASE_URL": tt.databaseURL} {
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
		})
	}
}
