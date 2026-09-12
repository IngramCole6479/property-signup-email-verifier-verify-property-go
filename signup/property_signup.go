package signup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"

	"github.com/example/property-signup-verifier/infrai"
)

type MaintenanceRequest struct {
	Category string `json:"category"`
	Status   string `json:"status"`
}

type TenantDocument struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type InspectionReminder struct {
	Kind  string    `json:"kind"`
	DueAt time.Time `json:"due_at"`
}

type PropertySignup struct {
	PropertyName        string               `json:"property_name"`
	ManagerEmail        string               `json:"manager_email"`
	MaintenanceRequests []MaintenanceRequest `json:"maintenance_requests"`
	TenantDocuments     []TenantDocument     `json:"tenant_documents"`
	InspectionReminders []InspectionReminder `json:"inspection_reminders"`
}

type Result struct {
	Status              string `json:"status"`
	MessageID           string `json:"message_id"`
	MaintenanceOpen     int    `json:"maintenance_open"`
	TenantDocumentsDue  int    `json:"tenant_documents_due"`
	InspectionReminders int    `json:"inspection_reminders"`
}

type EmailSender interface {
	SendEmail(context.Context, infrai.Email, string) (infrai.SendResult, error)
}

type Service struct {
	sender    EmailSender
	publicURL string
	newToken  func() (string, error)
}

func NewService(sender EmailSender, publicURL string) *Service {
	return &Service{sender: sender, publicURL: strings.TrimRight(publicURL, "/"), newToken: randomToken}
}

func (s *Service) Start(ctx context.Context, input PropertySignup) (Result, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(input.ManagerEmail))
	if err != nil || address.Address != strings.TrimSpace(input.ManagerEmail) {
		return Result{}, fmt.Errorf("manager_email must be a valid mailbox")
	}
	if strings.TrimSpace(input.PropertyName) == "" {
		return Result{}, fmt.Errorf("property_name is required")
	}

	token, err := s.newToken()
	if err != nil {
		return Result{}, fmt.Errorf("create verification token: %w", err)
	}
	link := s.publicURL + "/verify-email?token=" + token
	email := infrai.Email{
		To:      address.Address,
		Subject: "Verify your property manager email",
		HTML:    fmt.Sprintf("<p>Confirm access for <strong>%s</strong>.</p><p><a href=\"%s\">Verify email</a></p>", html.EscapeString(input.PropertyName), html.EscapeString(link)),
	}
	sent, err := s.sender.SendEmail(ctx, email, "property-signup-"+token)
	if err != nil {
		return Result{}, err
	}

	result := Result{Status: "pending_email_verification", MessageID: sent.MessageID, InspectionReminders: len(input.InspectionReminders)}
	for _, request := range input.MaintenanceRequests {
		if request.Status == "open" {
			result.MaintenanceOpen++
		}
	}
	for _, document := range input.TenantDocuments {
		if document.Status == "required" {
			result.TenantDocumentsDue++
		}
	}
	return result, nil
}

func randomToken() (string, error) {
	var value [24]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
