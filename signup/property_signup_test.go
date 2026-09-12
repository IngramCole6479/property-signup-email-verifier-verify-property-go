package signup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/example/property-signup-verifier/infrai"
)

type recordingSender struct {
	email infrai.Email
	key   string
}

func (s *recordingSender) SendEmail(_ context.Context, email infrai.Email, key string) (infrai.SendResult, error) {
	s.email, s.key = email, key
	return infrai.SendResult{MessageID: "msg_test_42"}, nil
}

func TestStartPropertySignup(t *testing.T) {
	tests := []struct {
		name     string
		input    PropertySignup
		wantOpen int
		wantDocs int
		wantErr  bool
	}{
		{
			name: "valid signup becomes pending and summarizes work",
			input: PropertySignup{
				PropertyName:        "Harbor Court",
				ManagerEmail:        "manager@example.com",
				MaintenanceRequests: []MaintenanceRequest{{Category: "plumbing", Status: "open"}, {Category: "paint", Status: "closed"}},
				TenantDocuments:     []TenantDocument{{Name: "lease", Status: "required"}, {Name: "insurance", Status: "received"}},
				InspectionReminders: []InspectionReminder{{Kind: "annual", DueAt: time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)}},
			},
			wantOpen: 1,
			wantDocs: 1,
		},
		{name: "invalid manager email stops before delivery", input: PropertySignup{PropertyName: "Harbor Court", ManagerEmail: "manager-at-example"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			service := NewService(sender, "http://localhost:8080")
			service.newToken = func() (string, error) { return "fixed-token", nil }
			got, err := service.Start(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil || sender.email.To != "" {
					t.Fatalf("expected validation error before email, got err=%v email=%+v", err, sender.email)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != "pending_email_verification" || got.MaintenanceOpen != tt.wantOpen || got.TenantDocumentsDue != tt.wantDocs {
				t.Fatalf("unexpected decision: %+v", got)
			}
			if sender.email.To != tt.input.ManagerEmail || !strings.Contains(sender.email.HTML, "/verify-email?token=fixed-token") {
				t.Fatalf("verification email boundary mismatch: %+v", sender.email)
			}
			if sender.key != "property-signup-fixed-token" || got.MessageID != "msg_test_42" {
				t.Fatalf("delivery identity mismatch: key=%q result=%+v", sender.key, got)
			}
		})
	}
}
