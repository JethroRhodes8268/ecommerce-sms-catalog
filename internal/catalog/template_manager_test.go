package catalog

import (
	"context"
	"encoding/json"
	"testing"
)

type recordingCreator struct {
	signatureCalls int
	templateCalls  int
}

func (r *recordingCreator) CreateSignature(_ context.Context, _ json.RawMessage, _ string) (json.RawMessage, error) {
	r.signatureCalls++
	return json.RawMessage(`{"id":"sig_42"}`), nil
}

func (r *recordingCreator) CreateTemplate(_ context.Context, _ json.RawMessage, _ string) (json.RawMessage, error) {
	r.templateCalls++
	return json.RawMessage(`{"id":"tpl_42"}`), nil
}

func TestManagerApprovalDecision(t *testing.T) {
	tests := []struct {
		name           string
		request        ApplyRequest
		wantState      string
		wantError      bool
		wantSignatures int
		wantTemplates  int
	}{
		{
			name:      "unapproved checkout copy is held",
			request:   ApplyRequest{Lifecycle: "checkout", Approved: false},
			wantState: "held_for_approval",
		},
		{
			name: "approved fulfillment copy is applied",
			request: ApplyRequest{
				Lifecycle: "fulfillment",
				Approved:  true,
				Signature: json.RawMessage(`{"name":"Northwind"}`),
				Template:  json.RawMessage(`{"name":"fulfillment-v2","body":"Order dispatched","variables":{}}`),
			},
			wantState:      "applied",
			wantSignatures: 1,
			wantTemplates:  1,
		},
		{
			name: "approved copy with invalid signature field is rejected",
			request: ApplyRequest{
				Lifecycle: "fulfillment",
				Approved:  true,
				Signature: json.RawMessage(`{"signature":"Northwind"}`),
				Template:  json.RawMessage(`{"name":"fulfillment-v2","body":"Order dispatched"}`),
			},
			wantError: true,
		},
		{
			name: "approved copy with invalid template fields is rejected",
			request: ApplyRequest{
				Lifecycle: "fulfillment",
				Approved:  true,
				Signature: json.RawMessage(`{"name":"Northwind"}`),
				Template:  json.RawMessage(`{"name":"fulfillment-v2","content":"Order dispatched","template_vars":{}}`),
			},
			wantError: true,
		},
		{
			name: "approved copy without required template body is rejected",
			request: ApplyRequest{
				Lifecycle: "fulfillment",
				Approved:  true,
				Signature: json.RawMessage(`{"name":"Northwind"}`),
				Template:  json.RawMessage(`{"name":"fulfillment-v2"}`),
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creator := &recordingCreator{}
			result, err := NewManager(creator).Apply(context.Background(), test.request)
			if test.wantError && err == nil {
				t.Fatal("expected an error")
			}
			if !test.wantError && err != nil {
				t.Fatal(err)
			}
			if result.State != test.wantState {
				t.Fatalf("state = %q, want %q", result.State, test.wantState)
			}
			if creator.signatureCalls != test.wantSignatures || creator.templateCalls != test.wantTemplates {
				t.Fatalf("calls = (%d, %d), want (%d, %d)", creator.signatureCalls, creator.templateCalls, test.wantSignatures, test.wantTemplates)
			}
		})
	}
}
