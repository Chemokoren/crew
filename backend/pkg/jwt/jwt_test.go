package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func newTestManager() *Manager {
	return NewManager("test-secret-key-that-is-at-least-32-chars-long", 15, 7)
}

func TestNewManager(t *testing.T) {
	m := newTestManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.accessExpiry != 15*time.Minute {
		t.Errorf("accessExpiry = %v, want 15m", m.accessExpiry)
	}
	if m.refreshExpiry != 7*24*time.Hour {
		t.Errorf("refreshExpiry = %v, want 7d", m.refreshExpiry)
	}
}

func TestGenerateAndVerifyToken(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	phone := "+254712345678"
	role := types.RoleCrewUser

	pair, err := m.GenerateTokenPair(userID, phone, role, nil, nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Error("AccessToken and RefreshToken should be different")
	}

	// Verify access token
	claims, err := m.VerifyToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Phone != phone {
		t.Errorf("Phone = %v, want %v", claims.Phone, phone)
	}
	if claims.SystemRole != role {
		t.Errorf("SystemRole = %v, want %v", claims.SystemRole, role)
	}
	if claims.CrewMemberID != nil {
		t.Errorf("CrewMemberID should be nil, got %v", claims.CrewMemberID)
	}
	if claims.Issuer != "amy-mis" {
		t.Errorf("Issuer = %v, want amy-mis", claims.Issuer)
	}
}

func TestTokenWithOptionalClaims(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	crewID := uuid.New()
	saccoID := uuid.New()

	pair, err := m.GenerateTokenPair(userID, "+254700000000", types.RoleSaccoAdmin, &crewID, &saccoID)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	claims, err := m.VerifyToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	if claims.CrewMemberID == nil || *claims.CrewMemberID != crewID {
		t.Errorf("CrewMemberID = %v, want %v", claims.CrewMemberID, crewID)
	}
	if claims.SaccoID == nil || *claims.SaccoID != saccoID {
		t.Errorf("SaccoID = %v, want %v", claims.SaccoID, saccoID)
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	m := newTestManager()

	_, err := m.VerifyToken("invalid.token.string")
	if err == nil {
		t.Error("VerifyToken should fail with invalid token")
	}
}

func TestVerifyTokenWrongSecret(t *testing.T) {
	m1 := NewManager("secret-one-that-is-at-least-32-characters-long", 15, 7)
	m2 := NewManager("secret-two-that-is-at-least-32-characters-long", 15, 7)

	pair, err := m1.GenerateTokenPair(uuid.New(), "+254700000000", types.RoleCrewUser, nil, nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	_, err = m2.VerifyToken(pair.AccessToken)
	if err == nil {
		t.Error("VerifyToken should fail with wrong secret")
	}
}

func TestAllRoles(t *testing.T) {
	m := newTestManager()
	roles := []types.SystemRole{
		types.RoleSystemAdmin,
		types.RoleSaccoAdmin,
		types.RoleCrewUser,
		types.RoleLender,
		types.RoleInsurer,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			pair, err := m.GenerateTokenPair(uuid.New(), "+254700000000", role, nil, nil)
			if err != nil {
				t.Fatalf("GenerateTokenPair(%s): %v", role, err)
			}

			claims, err := m.VerifyToken(pair.AccessToken)
			if err != nil {
				t.Fatalf("VerifyToken: %v", err)
			}

			if claims.SystemRole != role {
				t.Errorf("SystemRole = %v, want %v", claims.SystemRole, role)
			}
		})
	}
}
