package litellm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestTeamMemberUpdateSendsRole(t *testing.T) {
	var captured map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key", true)
	d := schema.TestResourceDataRaw(t, resourceLiteLLMTeamMember().Schema, map[string]interface{}{
		"team_id":    "team-1",
		"user_id":    "user-1",
		"user_email": "user@example.com",
		"role":       "user",
	})
	d.SetId("team-1:user-1")

	if err := resourceLiteLLMTeamMemberUpdate(d, client); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	role, ok := captured["role"]
	if !ok {
		t.Fatalf("update payload missing role field: %v", captured)
	}
	if role != "user" {
		t.Fatalf("update payload sent role %v, want user", role)
	}
	if v, present := captured["max_budget_in_team"]; present {
		t.Fatalf("update payload must not carry unset max_budget_in_team, got %v", v)
	}
}

func TestTeamMemberCreateSendsBudgetOnlyWhenSet(t *testing.T) {
	cases := []struct {
		name       string
		budget     interface{}
		wantBudget interface{}
	}{
		{name: "unset", budget: nil, wantBudget: nil},
		{name: "set", budget: 10.0, wantBudget: 10.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured map[string]interface{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &captured)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			raw := map[string]interface{}{
				"team_id":    "team-1",
				"user_id":    "user-1",
				"user_email": "user@example.com",
				"role":       "user",
			}
			if tc.budget != nil {
				raw["max_budget_in_team"] = tc.budget
			}
			d := schema.TestResourceDataRaw(t, resourceLiteLLMTeamMember().Schema, raw)

			if err := resourceLiteLLMTeamMemberCreate(d, NewClient(srv.URL, "test-key", true)); err != nil {
				t.Fatalf("create failed: %v", err)
			}

			got, present := captured["max_budget_in_team"]
			if tc.wantBudget == nil && present {
				t.Fatalf("member_add payload must not carry unset max_budget_in_team, got %v", got)
			}
			if tc.wantBudget != nil && got != tc.wantBudget {
				t.Fatalf("member_add payload sent max_budget_in_team %v, want %v", got, tc.wantBudget)
			}
		})
	}
}
