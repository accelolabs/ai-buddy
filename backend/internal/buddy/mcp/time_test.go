package mcp

import (
	"encoding/json"
	"testing"
	"time"

	// Project packages.
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCurrentTimeForBothRoles(t *testing.T) {
	for _, role := range []string{users.RoleIntern, users.RoleMentor} {
		t.Run(role, func(t *testing.T) {
			h := NewHandler(nil, users.User{Role: role}, nil, nil)
			session, closeSession, err := h.OpenSession(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer closeSession()
			before := time.Now().Truncate(time.Second)
			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: "get_current_time", Arguments: map[string]any{},
			})
			if err != nil || result.IsError {
				t.Fatalf("current time: %+v %v", result, err)
			}
			data, err := json.Marshal(result.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			var output CurrentTimeResult
			if err := json.Unmarshal(data, &output); err != nil {
				t.Fatal(err)
			}
			now, err := time.Parse(time.RFC3339, output.CurrentTime)
			_, offset := now.Zone()
			if err != nil || output.Timezone != "Europe/Moscow" || offset != 3*60*60 ||
				now.Before(before) || now.After(time.Now()) {
				t.Fatalf("unexpected time: %+v %v", output, err)
			}
		})
	}
}
