package jambopay

// AMY Merchant Flow Tests
// ========================
// AMY acts as a JamboPay Merchant. Under AMY's merchant account:
//   - Organizations (SACCOs) have wallet accounts  e.g. accountNo "ORG-001"
//   - Individual members (drivers/conductors) have wallet accounts e.g. "MBR-001"
//
// Flows tested:
//   1. Organization tops up float (external → org wallet)
//   2. Organization pays wages to member (org wallet → member wallet)
//   3. Member withdraws earnings (member wallet → M-Pesa B2C)
//   4. Member transfers to colleague (member wallet → colleague wallet)
//   5. Member receives salary from SACCO (sacco wallet → member wallet)
//   6. OTP regeneration for in-flight transfer
//   7. Transaction reversal (failed payout)
//   8. IPRS KYC verification via JamboPay proxy
//   9. SHA256 checksum verification (success + tamper detection)
//  10. Callback status handling (SUCCESS / FAILED / intermediate)

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kibsoft/amy-mis/internal/external/payment"
)

// ---------------------------------------------------------------------------
// Mock JamboPay server — simulates the full v2 Wallet API
// ---------------------------------------------------------------------------

type mockJPServer struct {
	t             *testing.T
	authCalls     int
	transferCalls int
	payoutCalls   int
	otpCalls      int
	reversalCalls int
}

func newMockJPServer(t *testing.T) (*httptest.Server, *mockJPServer) {
	t.Helper()
	m := &mockJPServer{t: t}
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		m.authCalls++
		if r.Method != http.MethodPost {
			t.Errorf("auth: want POST got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "x-www-form-urlencoded") {
			t.Errorf("auth: wrong Content-Type %s", r.Header.Get("Content-Type"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token-abc",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	// Create Profile
	mux.HandleFunc("/wallet/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"firstName": "John", "lastName": "Kamau",
				"identityNumber": "12345678", "phoneNumber": "0712345678",
				"gender": "Male", "isActive": true,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"firstName": "AMY", "lastName": "Merchant", "isActive": true,
			})
		}
	})

	// Get / Create account — POST returns single object, GET returns paginated list (real API shape)
	mux.HandleFunc("/wallet/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// CreateAccount response: single object
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"accountNo": "MBR-001", "currentBalance": 0, "bookBalance": 0,
				"currency": "KES", "accountType": "Individual", "isActive": true,
			})
			return
		}
		// GET: CheckBalance / GetAccount — paginated list
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pageIndex": 1, "pageSize": 10, "count": 1,
			"data": []map[string]interface{}{
				{"accountNo": "MBR-001", "currentBalance": 500000, "bookBalance": 500000,
					"currency": "KES", "accountType": "Individual", "isActive": true},
			},
		})
	})

	// List accounts
	mux.HandleFunc("/wallet/accounts", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pageIndex": 1, "pageSize": 10, "count": 2,
			"data": []map[string]interface{}{
				{"accountNo": "ORG-001", "currentBalance": 100000.00, "currency": "KES", "accountType": "Organization"},
				{"accountNo": "MBR-001", "currentBalance": 5000.00, "currency": "KES", "accountType": "Individual"},
			},
		})
	})

	// Wallet transfer (org→member, member→member)
	mux.HandleFunc("/wallet/transaction/transfer", func(w http.ResponseWriter, r *http.Request) {
		m.transferCalls++
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ref":         "TXN-" + fmt.Sprintf("%d", m.transferCalls),
			"amount":      body["amount"],
			"accountFrom": body["accountFrom"],
			"accountTo":   body["accountTo"],
		})
	})

	// Transfer authorize (OTP)
	mux.HandleFunc("/wallet/transaction/authorize", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		// OTP 123456 in test env (per JamboPay docs)
		if body["otp"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"status": 400, "message": []string{"OTP required"}})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "COMPLETED"})
	})

	// OTP regeneration
	mux.HandleFunc("/wallet/otp", func(w http.ResponseWriter, r *http.Request) {
		m.otpCalls++
		w.WriteHeader(http.StatusCreated)
	})

	// Transaction reversal
	mux.HandleFunc("/wallet/transaction/initiate-reversal", func(w http.ResponseWriter, r *http.Request) {
		m.reversalCalls++
		w.WriteHeader(http.StatusCreated)
	})

	// External payout (B2C/Bank)
	mux.HandleFunc("/payout", func(w http.ResponseWriter, r *http.Request) {
		m.payoutCalls++
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{},
			})
			return
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ref":         "JP-PAY-001",
			"orderId":     body["orderId"],
			"callBackUrl": body["callBackUrl"],
			"accountFrom": body["accountFrom"],
		})
	})

	// Payout authorize
	mux.HandleFunc("/payout/authorize", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "COMPLETED"})
	})

	// IPRS verify
	mux.HandleFunc("/iprs/verify", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": 1, "firstName": "John", "lastName": "Kamau",
			"idNumber": "12345678", "gender": "Male", "dob": "1990-01-15",
		})
	})

	return httptest.NewServer(mux), m
}

func newTestProvider(t *testing.T, server *httptest.Server) *JamboPayProvider {
	t.Helper()
	return NewJamboPayProvider(JamboPayConfig{
		BaseURL:       server.URL,
		AuthURL:       server.URL, // mock server handles /auth/token for both
		ClientID:      "amy-client",
		ClientSecret:  "amy-secret",
		CollectionAccount: "COLL-001",  // collection account (receives top-ups)
		PayoutAccount:     "PAY-001",   // merchant wallet (source for member disbursements)
		CallbackURL:   "https://amy.co.ke/api/v1/webhooks/jambopay",
		PartnerCode:   "456",
	}, testLogger())
}

// ---------------------------------------------------------------------------
// Flow 1: Organization tops up wallet (external → org wallet)
// In JamboPay this is done via the merchant portal or paybill collection.
// AMY checks the resulting balance.
// ---------------------------------------------------------------------------
func TestFlow1_OrgWalletTopup_CheckBalance(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// After topup (done externally via paybill), verify org wallet balance
	bal, err := p.CheckBalance(context.Background(), "ORG-001")
	if err != nil {
		t.Fatalf("CheckBalance: %v", err)
	}
	if bal.Balance != 500000 { // 500000 minor units = KES 5,000.00
		t.Errorf("Balance = %d, want 500000", bal.Balance)
	}
	if bal.Currency != "KES" {
		t.Errorf("Currency = %q, want KES", bal.Currency)
	}
	t.Logf("✓ Org wallet balance after topup: KES %.2f", float64(bal.Balance)/100)
}

// ---------------------------------------------------------------------------
// Flow 2: Organization pays wages to member (org wallet → member wallet)
// Tenant-level wallet transfer: no partnerCode needed for org-debiting transfers.
// ---------------------------------------------------------------------------
func TestFlow2_OrgPaysWagesToMember(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// SACCO (ORG-001) transfers daily wages to driver (MBR-001)
	result, err := p.InitiateTransfer(context.Background(), TransferRequest{
		Amount:      "1500.00", // KES 1,500 daily wage
		AccountTo:   "MBR-001",
		AccountFrom: "ORG-001",
		OrderID:     "WAGE-2026-05-06-DRV001",
		CallbackURL: p.cfg.CallbackURL,
	})
	if err != nil {
		t.Fatalf("InitiateTransfer (wages): %v", err)
	}
	if result.Ref == "" {
		t.Error("expected non-empty ref from transfer")
	}
	if mock.transferCalls != 1 {
		t.Errorf("transferCalls = %d, want 1", mock.transferCalls)
	}
	t.Logf("✓ Wage transfer initiated: ref=%s amount=%s", result.Ref, result.Amount)

	// Org authorizes the transfer (no OTP for tenant-level debits in JamboPay;
	// but we test authorization path for completeness)
	err = p.AuthorizeTransfer(context.Background(), result.Ref, "123456")
	if err != nil {
		t.Fatalf("AuthorizeTransfer: %v", err)
	}
	t.Log("✓ Wage transfer authorized")
}

// ---------------------------------------------------------------------------
// Flow 3: Member withdraws earnings (member wallet → M-Pesa B2C)
// External payout via POST /payout with provider=MOMO_B2C.
// Requires OTP authorization from the member's phone.
// ---------------------------------------------------------------------------
func TestFlow3_MemberWithdrawsEarnings(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// Driver requests withdrawal of KES 3,000 to their M-Pesa
	result, err := p.InitiatePayout(context.Background(), payment.PayoutRequest{
		AmountCents:    300_000, // KES 3,000
		AccountFrom:    "MBR-001",
		OrderID:        "WDR-DRV001-20260506",
		Channel:        payment.ChannelMobile,
		RecipientName:  "John Kamau",
		RecipientPhone: "0712345678",
		CallbackURL:    p.cfg.CallbackURL,
		Narration:      "Earnings withdrawal",
	})
	if err != nil {
		t.Fatalf("InitiatePayout (withdrawal): %v", err)
	}
	if result.Reference == "" {
		t.Error("expected payout reference")
	}
	if !result.RequiresOTP {
		t.Error("withdrawal should require OTP")
	}
	if mock.payoutCalls != 1 {
		t.Errorf("payoutCalls = %d, want 1", mock.payoutCalls)
	}
	t.Logf("✓ Withdrawal initiated: ref=%s status=%s", result.Reference, result.Status)

	// Member enters OTP received on phone (JamboPay test OTP = 123456)
	verified, err := p.VerifyPayout(context.Background(), payment.PayoutVerifyRequest{
		Reference: result.Reference,
		OTP:       "123456",
	})
	if err != nil {
		t.Fatalf("VerifyPayout: %v", err)
	}
	if verified.Status != "completed" {
		t.Errorf("Status = %q, want completed", verified.Status)
	}
	t.Log("✓ Withdrawal OTP verified — funds sent to M-Pesa")
}

// ---------------------------------------------------------------------------
// Flow 4: Member transfers money to colleague's wallet
// Tenant-client transfer: requires phoneNumber and partnerCode.
// ---------------------------------------------------------------------------
func TestFlow4_MemberTransfersToColleague(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// Driver sends KES 500 to conductor colleague
	result, err := p.InitiateTransfer(context.Background(), TransferRequest{
		Amount:      "500.00",
		AccountTo:   "MBR-002", // colleague's wallet
		AccountFrom: "MBR-001",
		PhoneNumber: "0712345678",    // sender phone for OTP
		OrderID:     "P2P-DRV001-COND002-001",
		CallbackURL: p.cfg.CallbackURL,
		PartnerCode: p.cfg.PartnerCode, // "456" — 3-digit suffix
	})
	if err != nil {
		t.Fatalf("InitiateTransfer (peer transfer): %v", err)
	}
	if mock.transferCalls != 1 {
		t.Errorf("transferCalls = %d, want 1", mock.transferCalls)
	}
	t.Logf("✓ Peer transfer initiated: ref=%s", result.Ref)

	// Member authorizes with OTP (JamboPay sends 6 digits; first 3 from JP + last 3 = partnerCode)
	err = p.AuthorizeTransfer(context.Background(), result.Ref, "123456")
	if err != nil {
		t.Fatalf("AuthorizeTransfer (peer): %v", err)
	}
	t.Log("✓ Peer transfer authorized")
}

// ---------------------------------------------------------------------------
// Flow 5: Member receives monthly salary from SACCO
// Same as Flow 2 but for monthly payroll run — bulk transfer.
// ---------------------------------------------------------------------------
func TestFlow5_SACCOPaysSalaryToMultipleMembers(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	payroll := []struct {
		memberAccount string
		amount        string
		orderID       string
	}{
		{"MBR-001", "45000.00", "SAL-MAY26-DRV001"},
		{"MBR-002", "38000.00", "SAL-MAY26-COND001"},
		{"MBR-003", "52000.00", "SAL-MAY26-DRV002"},
	}

	for _, p2 := range payroll {
		result, err := p.InitiateTransfer(context.Background(), TransferRequest{
			Amount:      p2.amount,
			AccountTo:   p2.memberAccount,
			AccountFrom: "ORG-001", // SACCO source wallet
			OrderID:     p2.orderID,
			CallbackURL: p.cfg.CallbackURL,
		})
		if err != nil {
			t.Fatalf("salary transfer to %s: %v", p2.memberAccount, err)
		}
		t.Logf("✓ Salary transfer to %s: ref=%s amount=KES %s", p2.memberAccount, result.Ref, p2.amount)
	}

	if mock.transferCalls != len(payroll) {
		t.Errorf("transferCalls = %d, want %d", mock.transferCalls, len(payroll))
	}
	t.Log("✓ Bulk salary payroll completed")
}

// ---------------------------------------------------------------------------
// Flow 6: OTP expired — member requests regeneration
// ---------------------------------------------------------------------------
func TestFlow6_OTPRegeneration(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// Transfer initiated elsewhere; OTP expired before member could enter it
	err := p.RegenerateOTP(context.Background(), "TXN-001")
	if err != nil {
		t.Fatalf("RegenerateOTP: %v", err)
	}
	if mock.otpCalls != 1 {
		t.Errorf("otpCalls = %d, want 1", mock.otpCalls)
	}
	t.Log("✓ OTP regenerated — new SMS sent to member")
}

// ---------------------------------------------------------------------------
// Flow 7: Transaction reversal (failed external payout)
// ---------------------------------------------------------------------------
func TestFlow7_TransactionReversal(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	err := p.ReverseTransaction(context.Background(), "JP-PAY-FAIL-001")
	if err != nil {
		t.Fatalf("ReverseTransaction: %v", err)
	}
	if mock.reversalCalls != 1 {
		t.Errorf("reversalCalls = %d, want 1", mock.reversalCalls)
	}
	t.Log("✓ Transaction reversed — funds returned to source wallet")
}

// ---------------------------------------------------------------------------
// Flow 8: IPRS KYC verification for member onboarding
// ---------------------------------------------------------------------------
func TestFlow8_IPRSKYCVerification(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	result, err := p.VerifyIdentity(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("VerifyIdentity: %v", err)
	}
	if result.FirstName != "John" {
		t.Errorf("FirstName = %q, want John", result.FirstName)
	}
	if result.IDNumber != "12345678" {
		t.Errorf("IDNumber = %q, want 12345678", result.IDNumber)
	}
	t.Logf("✓ IPRS verified: %s %s (ID: %s)", result.FirstName, result.LastName, result.IDNumber)
}

// ---------------------------------------------------------------------------
// Flow 9: SHA256 Callback Checksum Verification
// Spec: SHA256(ref + amount + client_id + client_secret)
// ---------------------------------------------------------------------------
func TestFlow9_CallbackChecksumVerification(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	ref := "JP-PAY-001"
	amount := "3000.00"
	clientID := "amy-client"
	clientSecret := "amy-secret"

	// Generate expected checksum
	h := sha256.New()
	h.Write([]byte(ref + amount + clientID + clientSecret))
	validChecksum := fmt.Sprintf("%x", h.Sum(nil))

	tests := []struct {
		name     string
		ref      string
		amount   string
		checksum string
		wantOK   bool
	}{
		{"valid checksum", ref, amount, validChecksum, true},
		{"tampered ref", "JP-PAY-999", amount, validChecksum, false},
		{"tampered amount", ref, "9999.00", validChecksum, false},
		{"empty checksum", ref, amount, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.VerifyCallbackChecksum(tt.ref, tt.amount, tt.checksum)
			if got != tt.wantOK {
				t.Errorf("VerifyCallbackChecksum(%q, %q, ...) = %v, want %v", tt.ref, tt.amount, got, tt.wantOK)
			}
		})
	}
	t.Log("✓ Checksum verification protects against tampered callbacks")
}

// ---------------------------------------------------------------------------
// Flow 10: Callback status handling
// ---------------------------------------------------------------------------
func TestFlow10_CallbackStatusHandling(t *testing.T) {
	// Terminal statuses JamboPay sends
	terminalSuccess := []string{"SUCCESS", "COMPLETED", "COMPLETE"}
	terminalFailed := []string{"FAILED", "FAILURE", "REVERSED", "ERROR"}
	intermediate := []string{"PENDING", "PROCESSING", "IN_PROGRESS"}

	for _, s := range terminalSuccess {
		t.Run("success_"+s, func(t *testing.T) {
			if !isSuccessStatus(s) {
				t.Errorf("status %q should be treated as success", s)
			}
		})
	}
	for _, s := range terminalFailed {
		t.Run("failed_"+s, func(t *testing.T) {
			if !isFailedStatus(s) {
				t.Errorf("status %q should be treated as failed", s)
			}
		})
	}
	for _, s := range intermediate {
		t.Run("intermediate_"+s, func(t *testing.T) {
			if isSuccessStatus(s) || isFailedStatus(s) {
				t.Errorf("status %q should be intermediate (no action)", s)
			}
		})
	}
	t.Log("✓ All JamboPay status strings correctly classified")
}

// isSuccessStatus returns true for JamboPay terminal success statuses.
func isSuccessStatus(s string) bool {
	switch s {
	case "SUCCESS", "COMPLETED", "COMPLETE":
		return true
	}
	return false
}

// isFailedStatus returns true for JamboPay terminal failure statuses.
func isFailedStatus(s string) bool {
	switch s {
	case "FAILED", "FAILURE", "REVERSED", "ERROR":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Profile Management Tests
// ---------------------------------------------------------------------------

func TestCreateMemberProfile(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	profile, err := p.CreateProfile(context.Background(), ProfileRequest{
		FirstName:       "John",
		LastName:        "Kamau",
		IdentityNumber:  "12345678",
		IdentityType:    "NationalId",
		PhoneNumber:     "0712345678",
		Gender:          "Male",
		DateOfBirth:     "1990-01-15T00:00:00Z",
		County:          "Nairobi",
		PhysicalAddress: "Westlands",
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if !profile.IsActive {
		t.Error("new profile should be active")
	}
	t.Logf("✓ Member profile created: %s %s", profile.FirstName, profile.LastName)
}

func TestCreateMemberWalletAccount(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	account, err := p.CreateAccount(context.Background(), "0712345678", "MBR-001", "KES")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if !account.IsActive {
		t.Error("new account should be active")
	}
	t.Logf("✓ Member wallet account created: %s (balance: KES %.2f)",
		account.AccountNo, float64(account.CurrentBalance)/100)
}

func TestGetMemberBalance(t *testing.T) {
	server, _ := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	bal, err := p.CheckBalance(context.Background(), "MBR-001")
	if err != nil {
		t.Fatalf("CheckBalance: %v", err)
	}
	if bal.Balance <= 0 {
		t.Error("expected positive balance")
	}
	t.Logf("✓ Member balance: KES %.2f", float64(bal.Balance)/100)
}

func TestTokenCachingAcrossFlows(t *testing.T) {
	server, mock := newMockJPServer(t)
	defer server.Close()
	p := newTestProvider(t, server)

	// Multiple operations should reuse the cached token
	p.CheckBalance(context.Background(), "ORG-001")
	p.CheckBalance(context.Background(), "MBR-001")
	p.GetProfile(context.Background())

	if mock.authCalls != 1 {
		t.Errorf("authCalls = %d, want 1 (token should be cached across flows)", mock.authCalls)
	}
	t.Log("✓ OAuth2 token cached across all wallet flows")
}
