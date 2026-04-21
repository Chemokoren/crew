package iprs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kibsoft/amy-mis/internal/external/identity"
)

// IPRSConfig holds configuration for the IPRS (Integrated Population Registration System) API.
type IPRSConfig struct {
	BaseURL             string `json:"base_url"`              // e.g. http://192.168.11.29:1550/api/v1/iprs
	AccessTokenEndpoint string `json:"access_token_endpoint"` // e.g. https://extenzia.jambopay.com:5101/connect/token
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
}

// IPRSProvider implements the identity.Provider interface using the IPRS API.
// Auth uses OAuth2 client_credentials with scope "iprs" via the JamboPay identity server.
type IPRSProvider struct {
	cfg    IPRSConfig
	client *http.Client
	logger *slog.Logger

	// Token cache
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

// NewIPRSProvider creates a new IPRS identity verification provider.
func NewIPRSProvider(cfg IPRSConfig, logger *slog.Logger) *IPRSProvider {
	return &IPRSProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *IPRSProvider) Name() string { return "iprs" }

// authenticate retrieves or refreshes the IPRS OAuth2 token.
func (p *IPRSProvider) authenticate(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	form := url.Values{
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"scope":         {"iprs"},
		"grant_type":    {"client_credentials"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.AccessTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build IPRS auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("IPRS auth failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IPRS auth returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode IPRS auth response: %w", err)
	}

	p.mu.Lock()
	p.token = tokenResp.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	p.mu.Unlock()

	return tokenResp.AccessToken, nil
}

// VerifyCitizen looks up citizen details by national ID number via the IPRS API.
// POST /citizen-details with Bearer token auth.
func (p *IPRSProvider) VerifyCitizen(ctx context.Context, req identity.VerifyRequest) (*identity.CitizenDetails, error) {
	p.logger.Info("verifying citizen via IPRS", slog.String("id_number", req.IDNumber))

	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, fmt.Errorf("IPRS auth: %w", err)
	}

	payload := map[string]string{
		"idNumber":     req.IDNumber,
		"serialNumber": req.SerialNumber,
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/citizen-details", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("build IPRS request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("IPRS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPRS returned %d", resp.StatusCode)
	}

	var iprsResp struct {
		FirstName    string `json:"firstName"`
		MiddleName   string `json:"middleName"`
		Surname      string `json:"surname"`
		Gender       string `json:"gender"`
		DateOfBirth  string `json:"dateOfBirth"`
		PlaceOfBirth string `json:"placeOfBirth"`
		Citizenship  string `json:"citizenship"`
		IDNumber     string `json:"idNumber"`
		SerialNumber string `json:"serialNumber"`
		Photo        string `json:"photo"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&iprsResp); err != nil {
		return nil, fmt.Errorf("decode IPRS response: %w", err)
	}

	p.logger.Info("IPRS verification successful",
		slog.String("id_number", req.IDNumber),
		slog.String("name", iprsResp.FirstName+" "+iprsResp.Surname),
	)

	return &identity.CitizenDetails{
		Provider:     p.Name(),
		IDNumber:     iprsResp.IDNumber,
		SerialNumber: iprsResp.SerialNumber,
		FirstName:    iprsResp.FirstName,
		MiddleName:   iprsResp.MiddleName,
		LastName:     iprsResp.Surname,
		Gender:       iprsResp.Gender,
		DateOfBirth:  iprsResp.DateOfBirth,
		PlaceOfBirth: iprsResp.PlaceOfBirth,
		Citizenship:  iprsResp.Citizenship,
		Photo:        iprsResp.Photo,
		Verified:     true,
	}, nil
}
