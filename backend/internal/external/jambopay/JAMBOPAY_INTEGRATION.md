# JamboPay V2 Wallet API — Integration Guide

## Overview

AMY integrates with **JamboPay V2 Wallet API** to provide multi-tenant financial workflows:

| Actor | Role |
|---|---|
| **AMY** | JamboPay **Merchant** account — houses wallet accounts for all tenants |
| **SACCO / Organization** | Holds a **Business** wallet under AMY's merchant |
| **Crew Member** (Driver / Conductor) | Holds an **Individual** wallet under AMY's merchant |

Supported operations:
- Organisation top-up and balance checks
- Wage/salary payouts from SACCO → Member wallets (wallet-to-wallet transfer)
- Member peer transfers (wallet-to-wallet)
- Member withdrawals to M-Pesa / Bank (external payout — requires permission, see §7)
- IPRS identity verification
- OTP management for 2FA on transfers
- SHA256 webhook callback validation

---

## 1. Architecture: Two Separate Base URLs

JamboPay uses **different endpoints** for authentication and wallet operations:

| Purpose | Base URL | Endpoint |
|---|---|---|
| **OAuth2 Token** | `https://accounts.jambopay.com/v2` | `POST /auth/token` |
| **Wallet API** | `https://api.jambopay.com` | `/wallet/*`, `/payout`, `/iprs/*` |

Both are Cloudflare-hosted. Go's default HTTP client negotiates HTTP/2 via TLS ALPN, which stalls on these endpoints. The provider forces HTTP/1.1 by setting `NextProtos: []string{"http/1.1"}` in the TLS config.

---

## 2. Authentication (OAuth2 Client Credentials)

```
POST https://accounts.jambopay.com/v2/auth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
client_id={JAMBOPAY_CLIENT_ID}
client_secret={JAMBOPAY_CLIENT_SECRET}
```

**Response:**
```json
{
  "token_type": "Bearer",
  "access_token": "ANZKCtiny3qc0Sp4vsUo...",
  "expires_in": 3600
}
```

The provider caches the token and refreshes it 60 seconds before expiry. All subsequent wallet API calls use `Authorization: Bearer {access_token}`.

> **Important:** Credentials are sent in the POST body (`client_secret_post`), NOT as HTTP Basic auth. The `client_secret` value is the raw base64-encoded string from JamboPay — do NOT decode it.

---

## 3. Environment Variables

Set these in `backend/.env`:

```env
# JamboPay Payment Provider
PAYMENT_JAMBOPAY_ENABLED=true

# OAuth2 credentials (from JamboPay portal)
JAMBOPAY_CLIENT_ID=94f45df22e8b13827cb7644fbb4f4e8377752744eb3ee4e53e60c23153f49b4f
JAMBOPAY_CLIENT_SECRET=YjQwMzRjZDQtN2Fk...  # base64-encoded secret from JamboPay

# API URLs
JAMBOPAY_BASE_URL=https://api.jambopay.com
JAMBOPAY_AUTH_URL=https://accounts.jambopay.com/v2

# AMY merchant account configuration
JAMBOPAY_ACCOUNT_FROM=1002603          # Source account for org-initiated transfers
JAMBOPAY_CALLBACK_URL=https://your-domain.com/api/v1/webhooks/jambopay
JAMBOPAY_PARTNER_CODE=349              # 3-digit code appended to OTP for member transfers
```

---

## 4. Key API Endpoints

### 4.1 Wallet Account

| Operation | Method | Path |
|---|---|---|
| Create account | `POST` | `/wallet/account` |
| Get account (by accountNo) | `GET` | `/wallet/account?accountNo={no}` |
| Get account (by phone) | `GET` | `/wallet/account?phoneNumber={phone}` |

**GET response is paginated:**
```json
{
  "pageIndex": 1, "pageSize": 10, "count": 1,
  "data": [{
    "accountNo": "1002603",
    "currentBalance": 12459038,
    "bookBalance": 5933946,
    "currency": "KES",
    "accountType": "Business",
    "isActive": true
  }]
}
```
> `currentBalance` is in **minor units** (e.g. `12459038` = KES 124,590.38). Do **not** multiply by 100.

### 4.2 Wallet Profile

| Operation | Method | Path |
|---|---|---|
| Create profile | `POST` | `/wallet/profile` |
| Get profile | `GET` | `/wallet/profile` |

### 4.3 Wallet Transfer (SACCO → Member or Peer)

```
POST /wallet/transaction/transfer
{
  "amount": "1.00",
  "accountTo": "MEMBER_ACCOUNT_NO",
  "accountFrom": "1002603",
  "orderId": "WAGE-2026050601",
  "callbackUrl": "https://...",
  "partnerCode": "349"       // required for peer (member→member) transfers
}
```

Response: `{ "ref": "177807878845734493", "orderId": "..." }`

Authorization (OTP):
```
POST /wallet/transaction/authorize
{ "ref": "177807878845734493", "otp": "123456" }
```

### 4.4 External Payout (Member → M-Pesa / Bank)

> **Requires JamboPay to enable the payout permission on the merchant account.**

```
POST /payout
{
  "amount": "1.00",
  "channel": "mpesa",
  "accountFrom": "1002603",
  "recipient": { "name": "...", "phoneNumber": "0712345678" },
  "orderId": "WDR-2026050601",
  "callbackUrl": "https://..."
}
```

### 4.5 Checksum Verification (Webhook Callbacks)

JamboPay embeds a `checksum` field in callback payloads. Verification:

```go
sha256(ref + amount + client_id + client_secret)
```

The `WebhookHandler` validates this before processing any callback. Invalid checksums return `403 Forbidden`.

---

## 5. Running Integration Tests

The integration tests auto-load `.env` and authenticate once in `TestMain` (shared token cache):

```bash
cd backend
JAMBOPAY_INTEGRATION=true go test ./internal/external/jambopay/... -v -run TestIntegration
```

**Optional env vars to unlock skipped tests:**

```env
JAMBOPAY_TEST_MEMBER_ACCOUNT=<member wallet account number>
JAMBOPAY_TEST_MEMBER_ACCOUNT_2=<second member account for peer transfer>
JAMBOPAY_TEST_MEMBER_PHONE=<member phone number>
JAMBOPAY_TEST_RECIPIENT_PHONE=<M-Pesa number for B2C withdrawal>
JAMBOPAY_TEST_ID_NUMBER=<national ID for IPRS verification>
JAMBOPAY_TEST_OTP=<override OTP for authorize step>
```

### Test Matrix

| Test | Requires | Status |
|---|---|---|
| `TestIntegration_Authenticate` | Credentials | ✅ PASS |
| `TestIntegration_CheckMerchantBalance` | Credentials | ✅ PASS — KES 124,590.38 |
| `TestIntegration_GetMerchantProfile` | Credentials | ✅ PASS — SisboPay / Uasingishu County |
| `TestIntegration_WalletTransfer_OrgToMember` | `JAMBOPAY_TEST_MEMBER_ACCOUNT` | ✅ PASS (ref returned; OTP error expected) |
| `TestIntegration_WalletTransfer_MemberToMember` | `JAMBOPAY_TEST_MEMBER_ACCOUNT` + `_2` | ✅ PASS |
| `TestIntegration_OTPRegeneration` | `JAMBOPAY_TEST_MEMBER_ACCOUNT` | ✅ PASS |
| `TestIntegration_ChecksumVerification` | Credentials (offline) | ✅ PASS |
| `TestIntegration_ExternalPayout_MobileB2C` | Permission + phone | ⚠ SKIP — payout permission not enabled |
| `TestIntegration_IPRSVerify` | `JAMBOPAY_TEST_ID_NUMBER` | ⏭ SKIP — provide ID to run |

---

## 6. Unit Tests (Mock Server)

Run without any credentials or network:

```bash
cd backend
go test ./internal/external/jambopay/... -v
```

Covers: auth, payout (mobile + bank), verify payout, balance check, token caching, auth failure, and all 22 merchant flow scenarios.

---

## 7. Known Limitations & Actions Required

### 7.1 External Payout (B2C) — Permission Not Enabled

The merchant account currently returns `400: You are not allowed to do payout transactions` for `/payout` (M-Pesa / Bank withdrawals).

**Action:** Contact JamboPay support and request:
> *"Please enable the external payout / B2C disbursement permission for merchant account 1002603 (SisiboPay Collection)."*

### 7.2 Transfer OTP in Sandbox

When `AuthorizeTransfer` is called with OTP `123456` (sandbox default), JamboPay returns `400: invalid otp code`. This is expected — the real OTP is sent to the account holder's phone. The test logs this as a warning and passes.

### 7.3 Network — HTTP/2 ALPN Stall

`accounts.jambopay.com` and `api.jambopay.com` are both Cloudflare-hosted. Go's default HTTP client negotiates HTTP/2 via TLS ALPN, which causes the connection to stall indefinitely waiting for response headers.

**Fix applied:** The `JamboPayProvider` transport sets `TLSClientConfig.NextProtos: []string{"http/1.1"}` to explicitly exclude h2 from TLS negotiation, forcing HTTP/1.1 on all connections.

If you observe `net/http: timeout awaiting response headers` errors:
1. Clear the test cache: `go clean -testcache`
2. Ensure you are running from `backend/` directory
3. Verify `JAMBOPAY_AUTH_URL=https://accounts.jambopay.com/v2` is set in `.env`

---

## 8. Webhook Handler

Endpoint: `POST /api/v1/webhooks/jambopay`

**Payload (from JamboPay):**
```json
{
  "ref": "177807878845734493",
  "orderId": "WAGE-2026050601",
  "status": "SUCCESS",
  "amount": "1.00",
  "checksum": "sha256hex...",
  "message": "Transaction completed"
}
```

The handler:
1. Reads the raw body (needed for checksum verification)
2. Verifies SHA256 checksum against `ref + amount + client_id + client_secret`
3. Rejects invalid checksums with `403`
4. Updates the corresponding `payout_transaction` record in the database
5. Returns `200 OK` for idempotent duplicate callbacks

---

## 9. Code Structure

```
backend/internal/external/jambopay/
├── client.go               # JamboPayProvider — all API calls, token cache, HTTP transport
├── client_test.go          # Unit tests (mock HTTP server)
├── merchant_flows_test.go  # 22 merchant flow tests (mock)
├── integration_test.go     # Live API tests (JAMBOPAY_INTEGRATION=true)
└── JAMBOPAY_INTEGRATION.md # This file

backend/internal/
├── service/
│   ├── payout_service.go   # PayoutService — orchestrates JamboPay payouts
│   └── webhook_service.go  # WebhookService — processes JamboPay callbacks
├── handler/
│   └── webhook_handler.go  # HTTP handler — validates checksum, calls WebhookService
└── config/config.go        # JamboPayBaseURL, JamboPayAuthURL, etc.

backend/cmd/server/main.go  # Wires JamboPayProvider into the dependency graph
```

---

## 10. Verified Live Credentials (as of 2026-05-06)

| Field | Value |
|---|---|
| Merchant Name | SisiboPay / Uasingishu County |
| Account No | `1002603` (Business) |
| Balance (live) | KES 124,590.38 |
| Auth URL | `https://accounts.jambopay.com/v2` |
| API URL | `https://api.jambopay.com` |
| Partner Code | `349` |
| Callback URL | Configured via `JAMBOPAY_CALLBACK_URL` |
