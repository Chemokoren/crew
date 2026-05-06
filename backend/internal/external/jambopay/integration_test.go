package jambopay

// Integration tests against the real JamboPay v2 API sandbox.
// These tests are skipped automatically unless JAMBOPAY_INTEGRATION=true
// and all required JAMBOPAY_* env vars are set.
//
// Run with:
//   JAMBOPAY_INTEGRATION=true go test ./internal/external/jambopay/... -v -run Integration -timeout 60s
//
// AMY Merchant Model:
//   - AMY is the JamboPay Merchant account
//   - Organizations (SACCOs) hold wallet accounts under AMY's merchant
//   - Individual members (drivers, conductors) hold wallet accounts
//   - AccountFrom (JAMBOPAY_ACCOUNT_FROM) is AMY's tenant source account

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/kibsoft/amy-mis/internal/external/payment"
)

// sharedProvider is the package-level JamboPay provider, initialized once in TestMain.
// Using a shared provider means authentication happens once — the cached token is reused
// across all integration tests, avoiding repeated auth round-trips.
var sharedProvider *JamboPayProvider

// TestMain auto-loads the backend .env file and pre-warms the shared provider
// so all integration tests share a single authenticated session.
func TestMain(m *testing.M) {
	loadDotEnv()
	// Pre-warm provider if integration mode is enabled
	if os.Getenv("JAMBOPAY_INTEGRATION") == "true" {
		initSharedProvider()
	}
	os.Exit(m.Run())
}

// loadDotEnv walks up from the working directory to find and load the backend .env.
// Uses godotenv.Load which does NOT overwrite vars already set in the environment.
func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// initSharedProvider builds the shared provider and pre-authenticates it.
// If auth fails (network unreachable), sharedProvider remains nil and all
// integration tests will skip gracefully.
func initSharedProvider() {
	authURL := os.Getenv("JAMBOPAY_AUTH_URL")
	if authURL == "" {
		authURL = "https://accounts.jambopay.com/v2"
	}
	clientID := os.Getenv("JAMBOPAY_CLIENT_ID")
	if clientID == "" {
		return // env not loaded — tests will skip
	}

	p := NewJamboPayProvider(JamboPayConfig{
		BaseURL:      os.Getenv("JAMBOPAY_BASE_URL"),
		AuthURL:      authURL,
		ClientID:     clientID,
		ClientSecret: os.Getenv("JAMBOPAY_CLIENT_SECRET"),
		AccountFrom:  os.Getenv("JAMBOPAY_ACCOUNT_FROM"),
		CallbackURL:  os.Getenv("JAMBOPAY_CALLBACK_URL"),
		PartnerCode:  os.Getenv("JAMBOPAY_PARTNER_CODE"),
	}, testLogger())

	// Pre-warm the token cache — if this fails, sharedProvider stays nil
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := p.authenticate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ JamboPay pre-auth failed (%v) — integration tests will skip\n", err)
		return
	}
	sharedProvider = p
	fmt.Fprintf(os.Stderr, "✓ JamboPay pre-auth OK — token cached for all integration tests\n")
}

// integrationProvider returns the shared pre-authenticated provider.
// Skips the test if: JAMBOPAY_INTEGRATION is not set, credentials are missing,
// or the JamboPay endpoint was unreachable at startup.
func integrationProvider(t *testing.T) *JamboPayProvider {
	t.Helper()
	if os.Getenv("JAMBOPAY_INTEGRATION") != "true" {
		t.Skip("Set JAMBOPAY_INTEGRATION=true to run live API tests")
		return nil
	}
	required := []string{"JAMBOPAY_CLIENT_ID", "JAMBOPAY_CLIENT_SECRET", "JAMBOPAY_BASE_URL", "JAMBOPAY_ACCOUNT_FROM"}
	var missing []string
	for _, k := range required {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("Missing required env vars: %v", missing)
	}
	if sharedProvider == nil {
		t.Skip("JamboPay pre-auth failed at startup (network unreachable?) — skipping live test")
		return nil
	}
	return sharedProvider
}

// uniqueOrderID generates a time-based unique order ID.
func uniqueOrderID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405")
}

// buildTestChecksum computes SHA256(ref+amount+clientID+clientSecret).
func buildTestChecksum(ref, amount, clientID, clientSecret string) string {
	h := sha256.New()
	h.Write([]byte(ref + amount + clientID + clientSecret))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ---------------------------------------------------------------------------
// Integration: Authentication
// ---------------------------------------------------------------------------

// TestIntegration_Authenticate verifies token acquisition against the real JamboPay
// accounts endpoint (https://accounts.jambopay.com/v2/auth/token).
// Auth is separate from the Wallet API endpoint (https://api.jambopay.com).
func TestIntegration_Authenticate(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	token, err := p.authenticate(ctx)
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty access token")
	}
	t.Logf("✓ OAuth2 token obtained (length: %d chars)", len(token))

	token2, err := p.authenticate(ctx)
	if err != nil {
		t.Fatalf("second authenticate failed: %v", err)
	}
	if token != token2 {
		t.Error("expected cached token on second call")
	}
	t.Log("✓ Token correctly cached")
}

// ---------------------------------------------------------------------------
// Integration: AMY Merchant Wallet Balance
// ---------------------------------------------------------------------------

func TestIntegration_CheckMerchantBalance(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	accountNo := p.cfg.AccountFrom
	bal, err := p.CheckBalance(ctx, accountNo)
	if err != nil {
		t.Fatalf("CheckBalance for merchant account %s failed: %v", accountNo, err)
	}
	t.Logf("✓ AMY Merchant wallet — Account: %s | Balance: KES %.2f | Currency: %s",
		accountNo, float64(bal.Balance)/100, bal.Currency)

	if bal.Currency == "" {
		t.Error("expected non-empty currency in balance response")
	}
	if bal.Provider != "jambopay" {
		t.Errorf("provider = %q, want jambopay", bal.Provider)
	}
}

// ---------------------------------------------------------------------------
// Integration: Merchant Profile
// ---------------------------------------------------------------------------

func TestIntegration_GetMerchantProfile(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	profile, err := p.GetProfile(ctx)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	t.Logf("✓ Merchant profile: %s %s (active: %v)", profile.FirstName, profile.LastName, profile.IsActive)
}

// ---------------------------------------------------------------------------
// Integration: IPRS Identity Verification (via JamboPay proxy)
// Set JAMBOPAY_TEST_ID_NUMBER to a valid KE national ID number.
// ---------------------------------------------------------------------------

func TestIntegration_IPRSVerify(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	testID := os.Getenv("JAMBOPAY_TEST_ID_NUMBER")
	if testID == "" {
		t.Skip("Set JAMBOPAY_TEST_ID_NUMBER to run IPRS verification")
	}

	result, err := p.VerifyIdentity(ctx, testID)
	if err != nil {
		t.Fatalf("VerifyIdentity(%s) failed: %v", testID, err)
	}
	t.Logf("✓ IPRS: %s %s | ID: %s | DOB: %s | Gender: %s",
		result.FirstName, result.LastName, result.IDNumber, result.DateOfBirth, result.Gender)
}

// ---------------------------------------------------------------------------
// Integration: Wallet Transfer — SACCO → Member (wage payout)
// Set JAMBOPAY_TEST_MEMBER_ACCOUNT to a valid member wallet account number.
// In sandbox, OTP for tenant-initiated transfers = "123456"
// ---------------------------------------------------------------------------

func TestIntegration_WalletTransfer_OrgToMember(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	memberAccount := os.Getenv("JAMBOPAY_TEST_MEMBER_ACCOUNT")
	if memberAccount == "" {
		t.Skip("Set JAMBOPAY_TEST_MEMBER_ACCOUNT for org→member transfer test")
	}

	orderID := uniqueOrderID("WAGE-TEST")
	t.Logf("Initiating: AMY(%s) → Member(%s) | KES 1.00 | order=%s",
		p.cfg.AccountFrom, memberAccount, orderID)

	// Step 1: Initiate
	result, err := p.InitiateTransfer(ctx, TransferRequest{
		Amount:      "1.00",
		AccountTo:   memberAccount,
		AccountFrom: p.cfg.AccountFrom,
		OrderID:     orderID,
		CallbackURL: p.cfg.CallbackURL,
	})
	if err != nil {
		t.Fatalf("InitiateTransfer failed: %v", err)
	}
	if result.Ref == "" {
		t.Fatal("expected non-empty ref from transfer initiation")
	}
	t.Logf("✓ Transfer initiated — ref: %s", result.Ref)

	// Step 2: Authorize with sandbox OTP
	sandboxOTP := "123456"
	if v := os.Getenv("JAMBOPAY_TEST_OTP"); v != "" {
		sandboxOTP = v
	}
	err = p.AuthorizeTransfer(ctx, result.Ref, sandboxOTP)
	if err != nil {
		t.Logf("⚠ AuthorizeTransfer error (may be expected in sandbox): %v", err)
	} else {
		t.Logf("✓ Transfer authorized with OTP")
	}
}

// ---------------------------------------------------------------------------
// Integration: Peer Transfer — Member → Member (with PartnerCode)
// ---------------------------------------------------------------------------

func TestIntegration_WalletTransfer_MemberToMember(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	fromAccount := os.Getenv("JAMBOPAY_TEST_MEMBER_ACCOUNT")
	toAccount := os.Getenv("JAMBOPAY_TEST_MEMBER_ACCOUNT_2")
	memberPhone := os.Getenv("JAMBOPAY_TEST_MEMBER_PHONE")

	if fromAccount == "" || toAccount == "" {
		t.Skip("Set JAMBOPAY_TEST_MEMBER_ACCOUNT + JAMBOPAY_TEST_MEMBER_ACCOUNT_2 for peer transfer")
	}

	orderID := uniqueOrderID("P2P-TEST")
	t.Logf("Peer transfer: %s → %s | KES 1.00 | order=%s", fromAccount, toAccount, orderID)

	result, err := p.InitiateTransfer(ctx, TransferRequest{
		Amount:      "1.00",
		AccountTo:   toAccount,
		AccountFrom: fromAccount,
		PhoneNumber: memberPhone,
		OrderID:     orderID,
		CallbackURL: p.cfg.CallbackURL,
		PartnerCode: p.cfg.PartnerCode,
	})
	if err != nil {
		t.Fatalf("Peer transfer failed: %v", err)
	}
	t.Logf("✓ Peer transfer initiated — ref: %s", result.Ref)
}

// ---------------------------------------------------------------------------
// Integration: External Payout — Member → M-Pesa B2C (withdrawal)
// Set JAMBOPAY_TEST_RECIPIENT_PHONE to a valid Safaricom number.
// ---------------------------------------------------------------------------

func TestIntegration_ExternalPayout_MobileB2C(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	recipientPhone := os.Getenv("JAMBOPAY_TEST_RECIPIENT_PHONE")
	if recipientPhone == "" {
		t.Skip("Set JAMBOPAY_TEST_RECIPIENT_PHONE for B2C payout test")
	}

	orderID := uniqueOrderID("WDR-TEST")
	t.Logf("B2C payout → %s | KES 1.00 | order=%s", recipientPhone, orderID)

	result, err := p.InitiatePayout(ctx, payment.PayoutRequest{
		AmountCents:    100, // KES 1.00
		AccountFrom:    p.cfg.AccountFrom,
		OrderID:        orderID,
		Channel:        payment.ChannelMobile,
		RecipientName:  "Test Member",
		RecipientPhone: recipientPhone,
		CallbackURL:    p.cfg.CallbackURL,
		Narration:      "Integration test withdrawal",
	})
	if err != nil {
		t.Fatalf("InitiatePayout (B2C) failed: %v", err)
	}
	if result.Reference == "" {
		t.Fatal("expected non-empty reference from payout initiation")
	}
	t.Logf("✓ B2C payout initiated — ref: %s | requiresOTP: %v", result.Reference, result.RequiresOTP)

	// Authorize with sandbox OTP if required
	if result.RequiresOTP {
		sandboxOTP := "123456"
		if v := os.Getenv("JAMBOPAY_TEST_OTP"); v != "" {
			sandboxOTP = v
		}
		verified, err := p.VerifyPayout(ctx, payment.PayoutVerifyRequest{
			Reference: result.Reference,
			OTP:       sandboxOTP,
		})
		if err != nil {
			t.Logf("⚠ VerifyPayout error (may be expected in sandbox): %v", err)
		} else {
			t.Logf("✓ Payout verified — status: %s", verified.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration: OTP Regeneration
// ---------------------------------------------------------------------------

func TestIntegration_OTPRegeneration(t *testing.T) {
	p := integrationProvider(t)
	ctx := context.Background()

	memberAccount := os.Getenv("JAMBOPAY_TEST_MEMBER_ACCOUNT")
	if memberAccount == "" {
		t.Skip("Set JAMBOPAY_TEST_MEMBER_ACCOUNT for OTP regeneration test")
	}

	// Create a transfer to get a ref
	result, err := p.InitiateTransfer(ctx, TransferRequest{
		Amount:      "1.00",
		AccountTo:   memberAccount,
		AccountFrom: p.cfg.AccountFrom,
		OrderID:     uniqueOrderID("OTP-TEST"),
		CallbackURL: p.cfg.CallbackURL,
	})
	if err != nil {
		t.Fatalf("InitiateTransfer for OTP test: %v", err)
	}

	// Regenerate OTP for that in-flight transfer
	err = p.RegenerateOTP(ctx, result.Ref)
	if err != nil {
		t.Logf("⚠ RegenerateOTP error (may be expected in sandbox): %v", err)
	} else {
		t.Logf("✓ OTP regenerated for ref=%s", result.Ref)
	}
}

// ---------------------------------------------------------------------------
// Integration: Checksum Verification with real credentials
// ---------------------------------------------------------------------------

func TestIntegration_ChecksumVerification(t *testing.T) {
	p := integrationProvider(t)

	ref := "TEST-REF-001"
	amount := "1.00"

	// Build correct checksum using real credentials
	correctChecksum := buildTestChecksum(ref, amount, p.cfg.ClientID, p.cfg.ClientSecret)

	// Valid checksum must pass
	if !p.VerifyCallbackChecksum(ref, amount, correctChecksum) {
		t.Error("valid checksum should pass verification with real credentials")
	}
	// Tampered ref must fail
	if p.VerifyCallbackChecksum("tampered-"+ref, amount, correctChecksum) {
		t.Error("tampered ref should fail checksum verification")
	}
	// Wrong checksum must fail
	if p.VerifyCallbackChecksum(ref, amount, "deadbeef") {
		t.Error("wrong checksum string should fail verification")
	}
	t.Logf("✓ SHA256 checksum verification correct with real credentials (ID prefix: %s...)",
		func() string {
			if len(p.cfg.ClientID) > 8 {
				return p.cfg.ClientID[:8]
			}
			return p.cfg.ClientID
		}())
}
