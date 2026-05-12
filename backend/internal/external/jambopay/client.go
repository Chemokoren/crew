package jambopay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kibsoft/amy-mis/internal/external/payment"
)

// JamboPayConfig holds configuration for the JamboPay v2 Wallet API.
type JamboPayConfig struct {
	BaseURL       string // Wallet API base URL  e.g. https://api.jambopay.com
	AuthURL       string // OAuth2 token URL     e.g. https://accounts.jambopay.com/v2
	ClientID      string // OAuth2 client ID
	ClientSecret  string // OAuth2 client secret (raw, as provided in credentials)
	CollectionAccount string // Collection account — receives incoming funds (WALLET_COLLECTION_ACCOUNT=1002603)
	PayoutAccount     string // Merchant wallet — source for disbursements to members (WALLET_MERCHANT_ACCOUNT=1002602)
	CallbackURL   string // Webhook URL JamboPay notifies on completion
	PartnerCode   string // 3-digit code appended to OTP for tenant-client transactions
}

// JamboPayProvider implements the payment.Provider interface using JamboPay v2 Wallet API.
//
// Supported operations:
//   - Token auth (POST /auth/token, client_credentials, x-www-form-urlencoded)
//   - Profile management (POST/GET /wallet/profile)
//   - Profile account management (POST/GET /wallet/account, GET /wallet/accounts)
//   - Wallet-to-wallet transfer (POST /wallet/transaction/transfer)
//   - Transfer authorization / OTP (POST /wallet/transaction/authorize, POST /wallet/otp)
//   - Transaction reversal (POST /wallet/transaction/initiate-reversal)
//   - Mobile collection / STK push (POST /checkout/express)
//   - Payout (POST /payout, POST /payout/authorize, GET /payout, GET /payout/{id})
//   - IPRS identity verification (POST /iprs/verify)
//   - Balance check (GET /wallet/account?accountNo=...)
type JamboPayProvider struct {
	cfg    JamboPayConfig
	client *http.Client
	logger *slog.Logger

	// Token cache — refreshed automatically before expiry
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

func NewJamboPayProvider(cfg JamboPayConfig, logger *slog.Logger) *JamboPayProvider {
	// JamboPay's accounts endpoint (accounts.jambopay.com) is Cloudflare-hosted
	// and advertises HTTP/2 via TLS ALPN. Go's http.Transport, even with
	// ForceAttemptHTTP2:false, will still use h2 if the server offers it in the
	// TLS handshake. The h2 connection then stalls waiting for response headers.
	// Fix: explicitly set NextProtos to ["http/1.1"] to prevent h2 negotiation.
	transport := &http.Transport{
		ForceAttemptHTTP2: false,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"http/1.1"}, // disable h2 ALPN — required for accounts.jambopay.com
		},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConns:          10,
	}
	return &JamboPayProvider{
		cfg:    cfg,
		client: &http.Client{Transport: transport, Timeout: 15 * time.Second},
		logger: logger,
	}
}

func (p *JamboPayProvider) Name() string { return "jambopay" }

// ===================================================================
// TOKEN MANAGEMENT
// ===================================================================

// authenticate retrieves or refreshes the JamboPay OAuth2 access token.
//
// JamboPay uses two separate base URLs:
//   - Auth:  https://accounts.jambopay.com/v2/auth/token  (POST, client_credentials, form body)
//   - API:   https://api.jambopay.com/...                 (Bearer token required)
//
// POST {AuthURL}/auth/token
// Content-Type: application/x-www-form-urlencoded
// Body: grant_type=client_credentials&client_id={id}&client_secret={secret}
func (p *JamboPayProvider) authenticate(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	authURL := p.cfg.AuthURL
	if authURL == "" {
		// Sensible default — the official JamboPay accounts endpoint
		authURL = "https://accounts.jambopay.com/v2"
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		authURL+"/auth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("jambopay auth: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jambopay auth returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode auth response: %w", err)
	}

	p.mu.Lock()
	p.token = tokenResp.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	p.mu.Unlock()

	return tokenResp.AccessToken, nil
}

func (p *JamboPayProvider) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, 0, err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		// === DEBUG: Log serialized request body ===
		p.logger.Debug("[HTTP] >>> request",
			slog.String("method", method),
			slog.String("url", p.cfg.BaseURL+path),
			slog.String("body", string(b)),
		)
		reqBody = bytes.NewReader(b)
	} else {
		p.logger.Debug("[HTTP] >>> request",
			slog.String("method", method),
			slog.String("url", p.cfg.BaseURL+path),
			slog.String("body", "<nil>"),
		)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.cfg.BaseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("jambopay request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// === DEBUG: Log raw response ===
	p.logger.Debug("[HTTP] <<< response",
		slog.String("url", p.cfg.BaseURL+path),
		slog.Int("status", resp.StatusCode),
		slog.String("body", string(respBody)),
	)

	return respBody, resp.StatusCode, nil
}

// ===================================================================
// PROFILE MANAGEMENT
// ===================================================================

// ProfileRequest is the request body for CreateProfile.
type ProfileRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	IdentityNumber  string `json:"identityNumber"`
	IdentityType    string `json:"identityType"` // "NationalId" or "Passport"
	PhoneNumber     string `json:"phoneNumber"`
	Gender          string `json:"gender"`
	DateOfBirth     string `json:"dateOfBirth"` // ISO 8601: "2022-08-23T13:29:55.295Z"
	County          string `json:"county"`
	PhysicalAddress string `json:"physicalAddress"`
	Email           string `json:"email,omitempty"`
}

// ProfileResponse mirrors the JamboPay profile shape.
type ProfileResponse struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	IdentityNumber  string `json:"identityNumber"`
	PhoneNumber     string `json:"phoneNumber"`
	Gender          string `json:"gender"`
	DateOfBirth     string `json:"dateOfBirth"`
	County          string `json:"county"`
	PhysicalAddress string `json:"physicalAddress"`
	IsActive        bool   `json:"isActive"`
}

// CreateProfile creates a new wallet holder profile.
// POST /wallet/profile
func (p *JamboPayProvider) CreateProfile(ctx context.Context, req ProfileRequest) (*ProfileResponse, error) {
	p.logger.Info("creating JamboPay profile", slog.String("phone", req.PhoneNumber))

	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/profile", req)
	if err != nil {
		return nil, fmt.Errorf("create profile request: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, parseJamboPayError("create profile", status, body)
	}
	var profile ProfileResponse
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}
	return &profile, nil
}

// GetProfile retrieves the tenant's own profile.
// GET /wallet/profile
func (p *JamboPayProvider) GetProfile(ctx context.Context) (*ProfileResponse, error) {
	body, status, err := p.doRequest(ctx, http.MethodGet, "/wallet/profile", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseJamboPayError("get profile", status, body)
	}
	var profile ProfileResponse
	_ = json.Unmarshal(body, &profile)
	return &profile, nil
}

// ===================================================================
// PROFILE ACCOUNT MANAGEMENT
// ===================================================================

// AccountResponse mirrors the JamboPay wallet account shape.
type AccountResponse struct {
	AccountNo      string           `json:"accountNo"`
	CurrentBalance float64          `json:"currentBalance"`
	Currency       string           `json:"currency"`
	AccountType    string           `json:"accountType"`
	IsActive       bool             `json:"isActive"`
	IsDefault      bool             `json:"isDefault"`
	Profile        *ProfileResponse `json:"profile,omitempty"`
}

// CreateAccount creates a wallet account for a profile holder.
// POST /wallet/account
// accountNo is the tenant account number to link to.
func (p *JamboPayProvider) CreateAccount(ctx context.Context, phoneNumber, accountNo, currency string) (*AccountResponse, error) {
	p.logger.Info("creating JamboPay account", slog.String("phone", phoneNumber), slog.String("account", accountNo))

	payload := map[string]string{
		"currency":    currency,
		"phoneNumber": phoneNumber,
		"accountNo":   accountNo,
	}
	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/account", payload)
	if err != nil {
		return nil, fmt.Errorf("create account request: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, parseJamboPayError("create account", status, body)
	}
	var account AccountResponse
	_ = json.Unmarshal(body, &account)
	return &account, nil
}

// GetAccount retrieves a wallet account by phone number or account number.
// GET /wallet/account?phoneNumber=...&accountNo=...
func (p *JamboPayProvider) GetAccount(ctx context.Context, phoneNumber, accountNo string) (*AccountResponse, error) {
	query := url.Values{}
	if phoneNumber != "" {
		query.Set("phoneNumber", phoneNumber)
	}
	if accountNo != "" {
		query.Set("accountNo", accountNo)
	}
	path := "/wallet/account"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	body, status, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseJamboPayError("get account", status, body)
	}
	var account AccountResponse
	_ = json.Unmarshal(body, &account)
	return &account, nil
}

// ===================================================================
// WALLET-TO-WALLET TRANSFERS (Internal — within JamboPay)
// ===================================================================

// TransferRequest is the body for POST /wallet/transaction/transfer (tenant-to-tenant-client).
type TransferRequest struct {
	Amount      string `json:"amount"`      // String decimal, e.g. "500.00"
	AccountTo   string `json:"accountTo"`   // Recipient account number
	AccountFrom string `json:"accountFrom"` // Source account number
	PhoneNumber string `json:"phoneNumber,omitempty"` // Required for tenant-client transfers
	OrderID     string `json:"orderId"`
	CallbackURL string `json:"callbackUrl"`
	PartnerCode string `json:"partnerCode,omitempty"` // 3-digit code; blank for tenant-debiting transfers
}

// TransferResponse is returned from POST /wallet/transaction/transfer.
type TransferResponse struct {
	Ref         string `json:"ref"`
	Amount      string `json:"amount"`
	AccountFrom string `json:"accountFrom"`
	AccountTo   string `json:"accountTo"`
	PartnerCode string `json:"partnerCode,omitempty"`
}

// InitiateTransfer initiates a wallet-to-wallet transfer.
// POST /wallet/transaction/transfer
// For tenant → tenant-client transfers: include PhoneNumber and PartnerCode.
// For tenant account debits: omit PhoneNumber and PartnerCode.
func (p *JamboPayProvider) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	p.logger.Info("initiating JamboPay wallet transfer",
		slog.String("account_to", req.AccountTo),
		slog.String("amount", req.Amount),
		slog.String("order_id", req.OrderID),
	)

	// Use defaults from config when not overridden
	if req.AccountFrom == "" {
		req.AccountFrom = p.cfg.CollectionAccount
	}
	if req.CallbackURL == "" {
		req.CallbackURL = p.cfg.CallbackURL
	}

	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/transaction/transfer", req)
	if err != nil {
		return nil, fmt.Errorf("transfer request: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, parseJamboPayError("initiate transfer", status, body)
	}

	var result TransferResponse
	_ = json.Unmarshal(body, &result)
	return &result, nil
}

// AuthorizeTransfer completes a pending transfer with OTP.
// POST /wallet/transaction/authorize
// OTP format: first 3 digits from JamboPay + 3-digit partner code.
func (p *JamboPayProvider) AuthorizeTransfer(ctx context.Context, ref, otp string) error {
	p.logger.Info("authorizing JamboPay transfer", slog.String("ref", ref))

	payload := map[string]string{"ref": ref, "otp": otp}
	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/transaction/authorize", payload)
	if err != nil {
		return fmt.Errorf("authorize transfer: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return parseJamboPayError("authorize transfer", status, body)
	}
	return nil
}

// RegenerateOTP re-triggers the OTP SMS for an in-progress transfer.
// POST /wallet/otp
func (p *JamboPayProvider) RegenerateOTP(ctx context.Context, ref string) error {
	p.logger.Info("regenerating JamboPay OTP", slog.String("ref", ref))

	payload := map[string]string{"ref": ref}
	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/otp", payload)
	if err != nil {
		return fmt.Errorf("regenerate OTP: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return parseJamboPayError("regenerate OTP", status, body)
	}
	_ = body
	return nil
}

// ReverseTransaction initiates a transaction reversal.
// POST /wallet/transaction/initiate-reversal
func (p *JamboPayProvider) ReverseTransaction(ctx context.Context, ref string) error {
	p.logger.Info("reversing JamboPay transaction", slog.String("ref", ref))

	payload := map[string]string{"ref": ref}
	body, status, err := p.doRequest(ctx, http.MethodPost, "/wallet/transaction/initiate-reversal", payload)
	if err != nil {
		return fmt.Errorf("reverse transaction: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return parseJamboPayError("reverse transaction", status, body)
	}
	_ = body
	return nil
}

// ===================================================================
// EXTERNAL PAYOUTS (M-Pesa B2C, Bank, Paybill/Till)
// ===================================================================

// InitiatePayout initiates a JamboPay external payout.
// POST /payout
// Implements payment.Provider.InitiatePayout.
func (p *JamboPayProvider) InitiatePayout(ctx context.Context, req payment.PayoutRequest) (*payment.PayoutResult, error) {
	p.logger.Info("initiating JamboPay payout",
		slog.String("channel", string(req.Channel)),
		slog.Int64("amount_cents", req.AmountCents),
		slog.String("order_id", req.OrderID),
	)

	amount := fmt.Sprintf("%.2f", float64(req.AmountCents)/100)

	// Use config defaults when not overridden by caller
	accountFrom := req.AccountFrom
	if accountFrom == "" {
		accountFrom = p.cfg.CollectionAccount
	}
	callbackURL := req.CallbackURL
	if callbackURL == "" {
		callbackURL = p.cfg.CallbackURL
	}

	// Build channel-specific payTo object
	payTo := map[string]string{}
	switch req.Channel {
	case payment.ChannelMobile: // MOMO_B2C
		payTo["accountRef"] = req.RecipientName
		payTo["accountNumber"] = req.RecipientPhone
	case payment.ChannelBank: // BANK
		payTo["accountNumber"] = req.BankAccount
		payTo["accountRef"] = req.RecipientName
		payTo["bankCode"] = req.BankCode
	case payment.ChannelPaybill: // MOMO_B2B (Paybill/Till)
		payTo["accountNumber"] = req.PaybillNumber
		payTo["accountRef"] = req.PaybillRef
	default:
		return nil, fmt.Errorf("unsupported payout channel: %s", req.Channel)
	}

	payload := map[string]interface{}{
		"amount":      amount,
		"accountFrom": accountFrom,
		"orderId":     req.OrderID,
		"provider":    string(req.Channel),
		"payTo":       payTo,
		"callBackUrl": callbackURL,
		"narration":   req.Narration,
	}

	body, status, err := p.doRequest(ctx, http.MethodPost, "/payout", payload)
	if err != nil {
		return nil, fmt.Errorf("payout request failed: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, parseJamboPayError("initiate payout", status, body)
	}

	var payoutResp struct {
		Ref         string `json:"ref"`
		OrderID     string `json:"orderId"`
		CallBackURL string `json:"callBackUrl"`
		AccountFrom string `json:"accountFrom"`
	}
	if err := json.Unmarshal(body, &payoutResp); err != nil {
		return nil, fmt.Errorf("decode payout response: %w", err)
	}

	return &payment.PayoutResult{
		Provider:    p.Name(),
		Reference:   payoutResp.Ref,
		OrderID:     payoutResp.OrderID,
		Status:      "pending_otp",
		RequiresOTP: true,
	}, nil
}

// VerifyPayout authorizes a pending payout with OTP.
// POST /payout/authorize
// OTP is the full 6-digit code (first 3 from JamboPay SMS + partner code).
func (p *JamboPayProvider) VerifyPayout(ctx context.Context, req payment.PayoutVerifyRequest) (*payment.PayoutResult, error) {
	p.logger.Info("verifying JamboPay payout", slog.String("ref", req.Reference))

	payload := map[string]string{
		"ref": req.Reference,
		"otp": req.OTP,
	}

	body, status, err := p.doRequest(ctx, http.MethodPost, "/payout/authorize", payload)
	if err != nil {
		return nil, fmt.Errorf("verify payout: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, parseJamboPayError("verify payout", status, body)
	}

	return &payment.PayoutResult{
		Provider:  p.Name(),
		Reference: req.Reference,
		Status:    "completed",
	}, nil
}

// GetPayout retrieves payout details by ID.
// GET /payout/{id}
func (p *JamboPayProvider) GetPayout(ctx context.Context, payoutID string) (map[string]interface{}, error) {
	body, status, err := p.doRequest(ctx, http.MethodGet, "/payout/"+payoutID, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseJamboPayError("get payout", status, body)
	}
	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)
	return result, nil
}

// ===================================================================
// BALANCE CHECK
// ===================================================================

// CheckBalance retrieves wallet account balance by account number.
// GET /wallet/account?accountNo={accountNo}
// Implements payment.Provider.CheckBalance.
func (p *JamboPayProvider) CheckBalance(ctx context.Context, accountNo string) (*payment.BalanceResult, error) {
	p.logger.Info("checking JamboPay balance", slog.String("account", accountNo))

	body, status, err := p.doRequest(ctx, http.MethodGet, "/wallet/account?accountNo="+accountNo, nil)
	if err != nil {
		return nil, fmt.Errorf("check balance failed: %w", err)
	}
	if status != http.StatusOK {
		return nil, parseJamboPayError("check balance", status, body)
	}

	// Real API response: paginated list { pageIndex, pageSize, count, data: [...] }
	// currentBalance is already in minor units (e.g. 12420168 = KES 124,201.68)
	var listResp struct {
		Count int `json:"count"`
		Data  []struct {
			AccountNo      string `json:"accountNo"`
			CurrentBalance int64  `json:"currentBalance"`
			BookBalance    int64  `json:"bookBalance"`
			Currency       string `json:"currency"`
			AccountType    string `json:"accountType"`
			IsActive       bool   `json:"isActive"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("decode balance response: %w", err)
	}
	if len(listResp.Data) == 0 {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}

	acct := listResp.Data[0]
	return &payment.BalanceResult{
		Provider: p.Name(),
		Balance:  acct.CurrentBalance, // already in minor units
		Currency: acct.Currency,
	}, nil
}

// ===================================================================
// MOBILE MONEY COLLECTION (STK Push — M-Pesa / Airtel)
// ===================================================================

// ExpressCheckoutRequest is the body for POST /checkout/express.
// This triggers an STK push (mobile money) or wallet-as-service checkout
// to collect money into the tenant's collection wallet.
//
// See: JamboPay v2 API Documentation — EXPRESS CHECKOUT / PAYMENT API
type ExpressCheckoutRequest struct {
	OrderID       string                      `json:"orderId"`       // Unique order ID (idempotency key)
	Amount        string                      `json:"amount"`        // String decimal, e.g. "500.00"
	CallbackURL   string                      `json:"callBackUrl"`   // Callback URL for result notification
	AccountTo     string                      `json:"accountTo"`     // Collection wallet account receiving funds
	Description   string                      `json:"description"`   // Transaction description
	ModeOfPayment string                      `json:"modeOfPayment"` // "MOBILE_MONEY" or "WALLET_AS_SERVICE"
	Provider      string                      `json:"provider"`      // "MPESA", "AIRTEL_MONEY", or "JAMBOPAY"
	Data          ExpressCheckoutDataMobile    `json:"data"`          // Mode-specific data (phone, serviceType, etc.)
}

// ExpressCheckoutDataMobile holds mode-specific fields for mobile money checkout.
type ExpressCheckoutDataMobile struct {
	PhoneNumber string `json:"phoneNumber,omitempty"` // Phone number to push STK to (mobile money)
	AccountNo   string `json:"accountNo,omitempty"`   // Sender account number (wallet-as-service)
	ServiceType string `json:"serviceType"`           // "TOPUP" or "MERCHANTPAYMENT"
}

// ExpressCheckoutResponse is returned from POST /checkout/express.
type ExpressCheckoutResponse struct {
	OrderID     string `json:"orderId"`
	Currency    string `json:"currency"`
	OrderAmount string `json:"orderAmount"`
	CallbackURL string `json:"callBackUrl"`
	Description string `json:"description"`
	Ref         string `json:"ref"`
	Checksum    string `json:"checksum"`
}

// MobileCollectionRequest is kept as an alias for backward compatibility.
// New callers should prefer ExpressCheckoutRequest directly.
type MobileCollectionRequest = ExpressCheckoutRequest

// MobileCollectionResponse is kept as an alias for backward compatibility.
type MobileCollectionResponse = ExpressCheckoutResponse

// InitiateMobileCollection triggers an express checkout / STK push to collect money
// from a mobile phone into the tenant's collection wallet.
// POST /checkout/express
//
// Supported modeOfPayment: "MOBILE_MONEY", "WALLET_AS_SERVICE"
// Supported providers: "MPESA", "AIRTEL_MONEY" (for mobile), "JAMBOPAY" (for wallet)
// Service types: "TOPUP" (fund wallet), "MERCHANTPAYMENT" (merchant payment)
func (p *JamboPayProvider) InitiateMobileCollection(ctx context.Context, req ExpressCheckoutRequest) (*ExpressCheckoutResponse, error) {
	p.logger.Info("initiating JamboPay express checkout (STK push)",
		slog.String("provider", req.Provider),
		slog.String("mode", req.ModeOfPayment),
		slog.String("phone", req.Data.PhoneNumber),
		slog.String("amount", req.Amount),
		slog.String("order_id", req.OrderID),
	)

	// Apply defaults from config
	if req.AccountTo == "" {
		req.AccountTo = p.cfg.CollectionAccount
	}
	if req.CallbackURL == "" {
		req.CallbackURL = p.cfg.CallbackURL
	}
	if req.ModeOfPayment == "" {
		req.ModeOfPayment = "MOBILE_MONEY"
	}
	if req.Data.ServiceType == "" {
		req.Data.ServiceType = "TOPUP"
	}

	// === DEBUG: Log full payload before sending ===
	payloadJSON, _ := json.MarshalIndent(req, "", "  ")
	p.logger.Info("[STK_PUSH] >>> OUTGOING PAYLOAD to /checkout/express",
		slog.String("url", p.cfg.BaseURL+"/checkout/express"),
		slog.String("callback_url", req.CallbackURL),
		slog.String("account_to", req.AccountTo),
		slog.String("payload", string(payloadJSON)),
	)

	body, status, err := p.doRequest(ctx, http.MethodPost, "/checkout/express", req)
	if err != nil {
		p.logger.Error("[STK_PUSH] >>> REQUEST FAILED", slog.String("error", err.Error()))
		return nil, fmt.Errorf("express checkout request: %w", err)
	}

	// === DEBUG: Log full response ===
	p.logger.Info("[STK_PUSH] <<< RESPONSE from /checkout/express",
		slog.Int("status_code", status),
		slog.String("response_body", string(body)),
	)

	if status != http.StatusCreated && status != http.StatusOK {
		p.logger.Error("[STK_PUSH] <<< NON-OK STATUS",
			slog.Int("status", status),
			slog.String("body", string(body)),
		)
		return nil, parseJamboPayError("express checkout", status, body)
	}

	var result ExpressCheckoutResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode express checkout response: %w", err)
	}

	p.logger.Info("[STK_PUSH] <<< PARSED RESPONSE",
		slog.String("ref", result.Ref),
		slog.String("order_id", result.OrderID),
		slog.String("callback_url_in_response", result.CallbackURL),
		slog.String("checksum", result.Checksum),
	)

	return &result, nil
}

// ExpressStatusResponse represents the status of an express checkout/STK push.
type ExpressStatusResponse struct {
	OrderID     string `json:"orderId"`
	Status      string `json:"status"`      // "PENDING", "SUCCESS", "FAILED", "CANCELLED"
	Amount      string `json:"amount"`
	Ref         string `json:"ref"`
	Description string `json:"description"`
}

// CheckExpressStatus queries JamboPay for the current status of an express checkout (STK push).
//
// Tries multiple approaches since JamboPay docs are sparse:
//  1. POST /wallet/transaction/ with orderId/ref in body
//  2. GET /wallet/transaction/{ref} (ref as path parameter)
//
// Uses no-redirect client + trailing slash to bypass nginx 301 → port 8002 issue.
func (p *JamboPayProvider) CheckExpressStatus(ctx context.Context, orderID string, jpRef ...string) (*ExpressStatusResponse, error) {
	ref := ""
	if len(jpRef) > 0 && jpRef[0] != "" {
		ref = jpRef[0]
	}

	p.logger.Info("[STK_POLL] checking transaction status",
		slog.String("order_id", orderID),
		slog.String("jp_ref", ref),
	)

	// Get auth token
	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, fmt.Errorf("GATEWAY_UNREACHABLE: authentication failed: %w", err)
	}

	// No-redirect client — trailing slash on /wallet/transaction/ is important
	noRedirectClient := &http.Client{
		Transport: p.client.Transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// ─── Strategy 1: POST /wallet/transaction/ with body ───
	postBody := map[string]string{"orderId": orderID}
	if ref != "" {
		postBody["ref"] = ref
	}
	bodyBytes, _ := json.Marshal(postBody)
	postURL := p.cfg.BaseURL + "/wallet/transaction/"

	postReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(bodyBytes))
	postReq.Header.Set("Authorization", "Bearer "+token)
	postReq.Header.Set("Content-Type", "application/json")

	p.logger.Debug("[STK_POLL] trying POST /wallet/transaction/",
		slog.String("url", postURL),
		slog.String("body", string(bodyBytes)),
	)

	postResp, postErr := noRedirectClient.Do(postReq)
	if postErr == nil {
		defer postResp.Body.Close()
		postRespBody, _ := io.ReadAll(postResp.Body)

		p.logger.Info("[STK_POLL] POST response",
			slog.Int("http_status", postResp.StatusCode),
			slog.String("body", string(postRespBody)),
		)

		if postResp.StatusCode == http.StatusOK || postResp.StatusCode == http.StatusCreated {
			if result := p.parseTransactionStatus(orderID, ref, postRespBody); result != nil {
				return result, nil
			}
		}
	}

	// ─── Strategy 2: GET /wallet/transaction/{ref} (ref as path param) ───
	if ref != "" {
		getURL := p.cfg.BaseURL + "/wallet/transaction/" + ref
		getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)

		p.logger.Debug("[STK_POLL] trying GET /wallet/transaction/{ref}",
			slog.String("url", getURL),
		)

		getResp, getErr := noRedirectClient.Do(getReq)
		if getErr == nil {
			defer getResp.Body.Close()
			getRespBody, _ := io.ReadAll(getResp.Body)

			p.logger.Info("[STK_POLL] GET /{ref} response",
				slog.Int("http_status", getResp.StatusCode),
				slog.String("body", string(getRespBody)),
			)

			if getResp.StatusCode == http.StatusOK {
				if result := p.parseTransactionStatus(orderID, ref, getRespBody); result != nil {
					return result, nil
				}
			}
		}
	}

	// ─── Strategy 3: GET /wallet/transaction/{orderId} ───
	getURL3 := p.cfg.BaseURL + "/wallet/transaction/" + orderID
	getReq3, _ := http.NewRequestWithContext(ctx, http.MethodGet, getURL3, nil)
	getReq3.Header.Set("Authorization", "Bearer "+token)

	p.logger.Debug("[STK_POLL] trying GET /wallet/transaction/{orderId}",
		slog.String("url", getURL3),
	)

	getResp3, getErr3 := noRedirectClient.Do(getReq3)
	if getErr3 == nil {
		defer getResp3.Body.Close()
		getRespBody3, _ := io.ReadAll(getResp3.Body)

		p.logger.Info("[STK_POLL] GET /{orderId} response",
			slog.Int("http_status", getResp3.StatusCode),
			slog.String("body", string(getRespBody3)),
		)

		if getResp3.StatusCode == http.StatusOK {
			if result := p.parseTransactionStatus(orderID, ref, getRespBody3); result != nil {
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("GATEWAY_UNREACHABLE: JamboPay does not support transaction status queries. Please confirm this payment manually if you received an M-Pesa confirmation SMS")
}

// parseTransactionStatus extracts status from a JamboPay transaction response body.
func (p *JamboPayProvider) parseTransactionStatus(orderID, ref string, body []byte) *ExpressStatusResponse {
	var txData map[string]interface{}
	if err := json.Unmarshal(body, &txData); err != nil {
		return nil
	}

	result := &ExpressStatusResponse{OrderID: orderID, Ref: ref}
	for _, key := range []string{"status", "transactionStatus", "orderStatus", "paymentStatus"} {
		if s, ok := txData[key].(string); ok && s != "" {
			result.Status = s
		}
	}
	if s, ok := txData["ref"].(string); ok {
		result.Ref = s
	}
	if s, ok := txData["amount"].(string); ok {
		result.Amount = s
	}

	// Check nested data array
	if dataArr, ok := txData["data"].([]interface{}); ok && len(dataArr) > 0 {
		if firstItem, ok := dataArr[0].(map[string]interface{}); ok {
			for _, key := range []string{"status", "transactionStatus"} {
				if s, ok := firstItem[key].(string); ok && s != "" {
					result.Status = s
				}
			}
			if s, ok := firstItem["ref"].(string); ok {
				result.Ref = s
			}
		}
	}

	p.logger.Info("[STK_POLL] parsed status",
		slog.String("order_id", orderID),
		slog.String("status", result.Status),
		slog.String("ref", result.Ref),
	)

	if result.Status != "" {
		return result
	}
	return nil // no status found — treat as unsupported
}

// GetTransaction retrieves a transaction record by reference or order ID.
// GET /wallet/transaction?ref=...&orderId=...
func (p *JamboPayProvider) GetTransaction(ctx context.Context, ref, orderID string) (map[string]interface{}, error) {
	query := url.Values{}
	if ref != "" {
		query.Set("ref", ref)
	}
	if orderID != "" {
		query.Set("orderId", orderID)
	}
	path := "/wallet/transaction/" // trailing slash avoids nginx 301 redirect
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	body, status, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseJamboPayError("get transaction", status, body)
	}
	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)
	return result, nil
}

// InitiateCollection implements payment.Provider.InitiateCollection.
func (p *JamboPayProvider) InitiateCollection(ctx context.Context, req payment.CollectionRequest) (*payment.CollectionResult, error) {
	// Map generic provider name to JamboPay provider constant
	jpProvider := "MPESA"
	switch req.Provider {
	case "AIRTEL_MONEY", "airtel":
		jpProvider = "AIRTEL_MONEY"
	}

	amount := fmt.Sprintf("%.2f", float64(req.AmountCents)/100)

	// === DEBUG: Log incoming collection request ===
	p.logger.Info("[COLLECTION] incoming request",
		slog.Int64("amount_cents", req.AmountCents),
		slog.String("amount_formatted", amount),
		slog.String("order_id", req.OrderID),
		slog.String("provider_input", req.Provider),
		slog.String("provider_mapped", jpProvider),
		slog.String("phone", req.PhoneNumber),
		slog.String("account_to_input", req.AccountTo),
		slog.String("callback_url_input", req.CallbackURL),
		slog.String("config_callback_url", p.cfg.CallbackURL),
		slog.String("config_collection_account", p.cfg.CollectionAccount),
	)

	ecReq := ExpressCheckoutRequest{
		OrderID:       req.OrderID,
		Amount:        amount,
		AccountTo:     req.AccountTo,
		Description:   req.Description,
		CallbackURL:   req.CallbackURL,
		ModeOfPayment: "MOBILE_MONEY",
		Provider:      jpProvider,
		Data: ExpressCheckoutDataMobile{
			PhoneNumber: req.PhoneNumber,
			ServiceType: "TOPUP",
		},
	}

	resp, err := p.InitiateMobileCollection(ctx, ecReq)
	if err != nil {
		p.logger.Error("[COLLECTION] STK push failed", slog.String("error", err.Error()))
		return nil, err
	}

	p.logger.Info("[COLLECTION] STK push initiated successfully",
		slog.String("ref", resp.Ref),
		slog.String("order_id", resp.OrderID),
		slog.String("callback_url_returned", resp.CallbackURL),
	)

	return &payment.CollectionResult{
		Provider:  p.Name(),
		Reference: resp.Ref,
		OrderID:   resp.OrderID,
		Status:    "pending",
	}, nil
}

// ==================================================================="
// IPRS IDENTITY VERIFICATION (via JamboPay proxy)
// ===================================================================

// IPRSResponse mirrors the JamboPay /iprs/verify response.
type IPRSResponse struct {
	ID           int    `json:"id"`
	DateOfBirth  string `json:"dob"`
	DateOfDeath  string `json:"dod,omitempty"`
	FirstName    string `json:"firstName"`
	Gender       string `json:"gender"`
	IDNumber     string `json:"idNumber"`
	IDType       int    `json:"idType"`
	LastName     string `json:"lastName"`
	MiddleName   string `json:"middleName"`
	SerialNumber string `json:"serialNumber"`
}

// VerifyIdentity verifies a national ID / passport via JamboPay's IPRS proxy.
// POST /iprs/verify
func (p *JamboPayProvider) VerifyIdentity(ctx context.Context, idNumber string) (*IPRSResponse, error) {
	p.logger.Info("verifying identity via JamboPay IPRS", slog.String("id_number", idNumber))

	payload := map[string]string{"idNumber": idNumber}
	body, status, err := p.doRequest(ctx, http.MethodPost, "/iprs/verify", payload)
	if err != nil {
		return nil, fmt.Errorf("IPRS verify: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, parseJamboPayError("IPRS verify", status, body)
	}

	var result IPRSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode IPRS response: %w", err)
	}
	return &result, nil
}

// ===================================================================
// CHECKSUM VERIFICATION
// ===================================================================

// VerifyCallbackChecksum validates a JamboPay callback payload checksum.
// JamboPay spec: SHA256(ref + amount + client_id + client_secret)
// Call this from the webhook handler before processing any callback.
func (p *JamboPayProvider) VerifyCallbackChecksum(ref, amount, checksum string) bool {
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		return true // Skip in unconfigured/dev environments
	}
	h := sha256.New()
	h.Write([]byte(ref + amount + p.cfg.ClientID + p.cfg.ClientSecret))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return expected == checksum
}

// ===================================================================
// BANK TRANSFER VERIFICATION
// ===================================================================

// VerifyBankTransfer checks a bank transfer reference.
// JamboPay does not currently support bank transfer verification.
// This method returns ErrNotImplemented, allowing the Manager to
// fall back to other providers or manual approval.
func (p *JamboPayProvider) VerifyBankTransfer(_ context.Context, _ payment.BankVerificationRequest) (*payment.BankVerificationResult, error) {
	return nil, payment.ErrNotImplemented
}

// ===================================================================
// HELPERS
// ===================================================================

// parseJamboPayError decodes a JamboPay error response.
// Error shape: {"status": int, "message": ["string"]}
func parseJamboPayError(op string, status int, body []byte) error {
	var errResp struct {
		Status  int      `json:"status"`
		Message []string `json:"message"`
	}
	_ = json.Unmarshal(body, &errResp)
	if len(errResp.Message) > 0 {
		return fmt.Errorf("jambopay %s error %d: %v", op, status, errResp.Message)
	}
	return fmt.Errorf("jambopay %s returned %d: %s", op, status, string(body))
}
