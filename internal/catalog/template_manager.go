package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Creator interface {
	CreateSignature(context.Context, json.RawMessage, string) (json.RawMessage, error)
	CreateTemplate(context.Context, json.RawMessage, string) (json.RawMessage, error)
}

type ApplyRequest struct {
	Lifecycle string          `json:"lifecycle"`
	Approved  bool            `json:"approved"`
	Signature json.RawMessage `json:"signature"`
	Template  json.RawMessage `json:"template"`
}

type ApplyResult struct {
	Lifecycle string          `json:"lifecycle"`
	State     string          `json:"state"`
	Signature json.RawMessage `json:"signature"`
	Template  json.RawMessage `json:"template"`
}

type Manager struct {
	creator Creator
}

func NewManager(creator Creator) *Manager {
	return &Manager{creator: creator}
}

func (m *Manager) Apply(ctx context.Context, request ApplyRequest) (ApplyResult, error) {
	if !validLifecycle(request.Lifecycle) {
		return ApplyResult{}, fmt.Errorf("lifecycle must be checkout, fulfillment, receipt, or order_update")
	}
	if !request.Approved {
		return ApplyResult{Lifecycle: request.Lifecycle, State: "held_for_approval"}, nil
	}
	if err := validatePayload("signature", request.Signature, []string{"name", "type", "proof_url", "remark", "vendor"}, []string{"name"}); err != nil {
		return ApplyResult{}, err
	}
	if err := validatePayload("template", request.Template, []string{"name", "body", "locale", "variables", "vendor", "message_type", "international"}, []string{"name", "body"}); err != nil {
		return ApplyResult{}, err
	}

	signature, err := m.creator.CreateSignature(ctx, request.Signature, stableKey(request.Lifecycle, "signature", request.Signature))
	if err != nil {
		return ApplyResult{}, err
	}
	template, err := m.creator.CreateTemplate(ctx, request.Template, stableKey(request.Lifecycle, "template", request.Template))
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Lifecycle: request.Lifecycle, State: "applied", Signature: signature, Template: template}, nil
}

func validLifecycle(value string) bool {
	switch value {
	case "checkout", "fulfillment", "receipt", "order_update":
		return true
	default:
		return false
	}
}

func validatePayload(kind string, value json.RawMessage, allowed, required []string) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(value, &object) != nil || object == nil {
		return fmt.Errorf("%s must be a JSON object", kind)
	}

	allowedFields := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedFields[field] = struct{}{}
	}
	for field := range object {
		if _, ok := allowedFields[field]; !ok {
			return fmt.Errorf("%s contains unsupported field %q", kind, field)
		}
	}
	for _, field := range required {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("%s requires field %q", kind, field)
		}
	}
	return nil
}

func stableKey(lifecycle, resource string, payload json.RawMessage) string {
	sum := sha256.Sum256(append([]byte(lifecycle+":"+resource+":"), payload...))
	return "sms-catalog-" + hex.EncodeToString(sum[:16])
}
