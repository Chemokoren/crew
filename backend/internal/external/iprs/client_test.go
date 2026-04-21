package iprs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/kibsoft/amy-mis/internal/external/identity"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestServers(t *testing.T) (iprsServer *httptest.Server, tokenServer *httptest.Server) {
	t.Helper()

	tokenServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token method = %s, want POST", r.Method)
		}
		r.ParseForm()
		if r.FormValue("scope") != "iprs" {
			t.Errorf("scope = %q, want iprs", r.FormValue("scope"))
		}
		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", r.FormValue("grant_type"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "iprs-test-token",
			"expires_in":   3600,
		})
	}))

	iprsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer iprs-test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["idNumber"] == "00000000" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"firstName":    "Jane",
			"middleName":   "Wanjiku",
			"surname":      "Kamau",
			"gender":       "Female",
			"dateOfBirth":  "1990-05-15",
			"placeOfBirth": "Nairobi",
			"citizenship":  "Kenyan",
			"idNumber":     body["idNumber"],
			"serialNumber": body["serialNumber"],
			"photo":        "base64encodedphoto==",
		})
	}))

	return iprsServer, tokenServer
}

func TestIPRSName(t *testing.T) {
	p := NewIPRSProvider(IPRSConfig{}, testLogger())
	if p.Name() != "iprs" {
		t.Errorf("Name = %q, want iprs", p.Name())
	}
}

func TestIPRSVerifyCitizenSuccess(t *testing.T) {
	iprsServer, tokenServer := newTestServers(t)
	defer iprsServer.Close()
	defer tokenServer.Close()

	p := NewIPRSProvider(IPRSConfig{
		BaseURL:             iprsServer.URL,
		AccessTokenEndpoint: tokenServer.URL,
		ClientID:            "test-client",
		ClientSecret:        "test-secret",
	}, testLogger())

	result, err := p.VerifyCitizen(context.Background(), identity.VerifyRequest{
		IDNumber:     "12345678",
		SerialNumber: "SER001",
	})
	if err != nil {
		t.Fatalf("VerifyCitizen: %v", err)
	}
	if !result.Verified {
		t.Error("should be verified")
	}
	if result.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want Jane", result.FirstName)
	}
	if result.MiddleName != "Wanjiku" {
		t.Errorf("MiddleName = %q, want Wanjiku", result.MiddleName)
	}
	if result.LastName != "Kamau" {
		t.Errorf("LastName = %q, want Kamau", result.LastName)
	}
	if result.Gender != "Female" {
		t.Errorf("Gender = %q, want Female", result.Gender)
	}
	if result.DateOfBirth != "1990-05-15" {
		t.Errorf("DateOfBirth = %q, want 1990-05-15", result.DateOfBirth)
	}
	if result.Citizenship != "Kenyan" {
		t.Errorf("Citizenship = %q, want Kenyan", result.Citizenship)
	}
	if result.Photo != "base64encodedphoto==" {
		t.Errorf("Photo = %q, want base64encodedphoto==", result.Photo)
	}
	if result.Provider != "iprs" {
		t.Errorf("Provider = %q, want iprs", result.Provider)
	}
}

func TestIPRSVerifyCitizenNotFound(t *testing.T) {
	iprsServer, tokenServer := newTestServers(t)
	defer iprsServer.Close()
	defer tokenServer.Close()

	p := NewIPRSProvider(IPRSConfig{
		BaseURL: iprsServer.URL, AccessTokenEndpoint: tokenServer.URL,
		ClientID: "test", ClientSecret: "test",
	}, testLogger())

	_, err := p.VerifyCitizen(context.Background(), identity.VerifyRequest{
		IDNumber: "00000000",
	})
	if err == nil {
		t.Error("should fail for unknown ID number")
	}
}

func TestIPRSAuthFailure(t *testing.T) {
	badTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer badTokenServer.Close()

	p := NewIPRSProvider(IPRSConfig{
		BaseURL: "http://unused", AccessTokenEndpoint: badTokenServer.URL,
		ClientID: "bad", ClientSecret: "bad",
	}, testLogger())

	_, err := p.VerifyCitizen(context.Background(), identity.VerifyRequest{
		IDNumber: "12345678",
	})
	if err == nil {
		t.Error("should fail on auth failure")
	}
}

func TestIPRSTokenCaching(t *testing.T) {
	tokenCalls := 0

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached", "expires_in": 3600,
		})
	}))
	defer tokenServer.Close()

	iprsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"firstName": "A", "surname": "B", "idNumber": "X",
		})
	}))
	defer iprsServer.Close()

	p := NewIPRSProvider(IPRSConfig{
		BaseURL: iprsServer.URL, AccessTokenEndpoint: tokenServer.URL,
		ClientID: "c", ClientSecret: "s",
	}, testLogger())

	p.VerifyCitizen(context.Background(), identity.VerifyRequest{IDNumber: "1"})
	p.VerifyCitizen(context.Background(), identity.VerifyRequest{IDNumber: "2"})

	if tokenCalls != 1 {
		t.Errorf("Token fetched %d times, want 1", tokenCalls)
	}
}
