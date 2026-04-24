package session

import (
	"testing"
	"time"
)

func TestData_SetAndGetInput(t *testing.T) {
	d := &Data{
		SessionID:    "test-001",
		MSISDN:       "+254712345678",
		CurrentState: StateInit,
		Language:     "en",
		CreatedAt:    time.Now(),
	}

	// Test initial state (nil map)
	if got := d.GetInput("amount"); got != "" {
		t.Errorf("GetInput on nil map = %q, want empty", got)
	}

	// Test set and get
	d.SetInput("amount", "500")
	if got := d.GetInput("amount"); got != "500" {
		t.Errorf("GetInput after SetInput = %q, want %q", got, "500")
	}

	// Test multiple inputs
	d.SetInput("pin", "1234")
	if got := d.GetInput("pin"); got != "1234" {
		t.Errorf("GetInput('pin') = %q, want %q", got, "1234")
	}
	if got := d.GetInput("amount"); got != "500" {
		t.Errorf("GetInput('amount') = %q, want %q, shouldn't be overwritten", got, "500")
	}

	// Test clear
	d.ClearInputs()
	if got := d.GetInput("amount"); got != "" {
		t.Errorf("GetInput after ClearInputs = %q, want empty", got)
	}
}

func TestData_StateTransitions(t *testing.T) {
	states := []State{
		StateInit, StateMainMenu, StateCheckBalance,
		StateWithdraw, StateWithdrawAmount, StateWithdrawConfirm, StateWithdrawPIN,
		StateEarnings, StateLastPayment, StateLoanStatus,
		StateRegister, StateRegisterName, StateRegisterNationalID,
		StateRegisterRole, StateRegisterPIN, StateRegisterPINConfirm, StateRegisterConfirm,
		StateLanguageSelect, StateEnd,
	}

	for _, state := range states {
		d := &Data{CurrentState: state}
		if d.CurrentState != state {
			t.Errorf("State should be %q, got %q", state, d.CurrentState)
		}
	}
}
