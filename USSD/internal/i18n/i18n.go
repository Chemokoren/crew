// Package i18n provides internationalization support for USSD menus.
// USSD targets feature phone users across Kenya, so multi-language
// support (English + Swahili) is essential.
//
// Text is kept short (< 160 chars per screen) to comply with USSD constraints.
package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// Translator provides thread-safe multi-language text lookup.
type Translator struct {
	mu       sync.RWMutex
	messages map[string]map[string]string // lang -> key -> message
	fallback string
}

// NewTranslator creates a new translator with default messages loaded.
func NewTranslator(defaultLang string) *Translator {
	t := &Translator{
		messages: make(map[string]map[string]string),
		fallback: defaultLang,
	}

	// Load built-in translations
	t.loadEnglish()
	t.loadSwahili()

	return t
}

// T returns the translated message for the given language and key.
// Falls back to the default language if the key is not found.
func (t *Translator) T(lang, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if lang == "" {
		lang = t.fallback
	}

	// Try requested language
	if msgs, ok := t.messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// Fallback to default language
	if msgs, ok := t.messages[t.fallback]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// Key not found — return the key itself for debugging
	return fmt.Sprintf("[%s]", key)
}

// SetMessage sets a single message for a language.
func (t *Translator) SetMessage(lang, key, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.messages[lang]; !ok {
		t.messages[lang] = make(map[string]string)
	}
	t.messages[lang][key] = message
}

// SupportedLanguages returns a list of available language codes.
func (t *Translator) SupportedLanguages() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	langs := make([]string, 0, len(t.messages))
	for lang := range t.messages {
		langs = append(langs, lang)
	}
	return langs
}

// HasLanguage checks if a language is available.
func (t *Translator) HasLanguage(lang string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.messages[lang]
	return ok
}

// --- English Messages ---

func (t *Translator) loadEnglish() {
	en := map[string]string{
		// Main menu
		"menu.welcome":       "Welcome to CrewPay",
		"menu.check_balance": "Check Balance",
		"menu.withdraw":      "Withdraw",
		"menu.earnings":      "My Earnings",
		"menu.last_payment":  "Last Payment",
		"menu.loan_status":   "Loans",
		"menu.my_profile":    "My Profile",
		"menu.register":      "Register",
		"menu.language":      "Language",
		"menu.exit":          "Exit",
		"menu.back":          "Back to Menu",

		// Balance
		"balance.result": "Your balance: %s",

		// Withdraw
		"withdraw.enter_amount":     "Available: %s\nEnter amount to withdraw\n0. Back",
		"withdraw.invalid_amount":   "Invalid amount. Enter numbers only\n0. Back",
		"withdraw.confirm":          "Withdraw %s?\n1. Confirm\n2. Cancel",
		"withdraw.confirm_options":  "1. Confirm\n2. Cancel",
		"withdraw.enter_pin":        "Enter your transaction PIN",
		"withdraw.invalid_pin":      "Invalid PIN. Enter 4-6 digits",
		"withdraw.wrong_pin":        "Wrong PIN. Try again",
		"withdraw.no_pin_set":       "You have not set a transaction PIN.\nGo to My Profile (6) to set one.\n0. Back to Menu",
		"withdraw.success":          "Withdrawal of %s initiated.\nRef: %s\nYou will receive M-Pesa shortly.",
		"withdraw.insufficient_funds": "Insufficient balance for this withdrawal.",

		// Earnings
		"earnings.menu":           "My Earnings\n1. Today\n2. This Week\n3. This Month\n0. Back",
		"earnings.daily_result":   "Today's Earnings\nTotal: %s\nAssignments: %d",
		"earnings.weekly_result":  "This Week's Earnings\nTotal: %s",
		"earnings.monthly_result": "This Month's Earnings\nTotal: %s",

		// Last Payment
		"payment.no_transactions": "No transactions found.",
		"payment.last_result":     "Last Transaction\nType: %s\nAmount: %s\nDate: %s\nRef: %s",

		// Loans
		"loan.menu":           "Loans\n1. Check Status\n2. Apply for Loan\n3. Score Details\n0. Back",
		"loan.no_loans":       "You have no active loans.",
		"loan.status_result":  "Loan: %s\nStatus: %s",
		"loan.score_details":  "Credit Score: %d (%s)\n\nTop Factors:%s\n\nTips:%s",
		"loan.low_score":      "Your credit score (%d) is too low to apply for a loan.\nMinimum required: %d\n0. Back to Menu",
		"loan.tier_info":      "Your Loan Tier: %s\nCredit Score: %d\nMax Loan: KES %.0f\nInterest: %.0f%%\nMax Tenure: %d days\n\nEnter loan amount (KES)\n0. Back",
		"loan.enter_amount":   "Enter loan amount (KES)\n0. Back",
		"loan.invalid_amount": "Invalid amount. Enter numbers only\n0. Back",
		"loan.select_tenure":  "Select repayment period\n1. 7 days\n2. 14 days\n3. 30 days\n0. Back",
		"loan.select_tenure_14": "Select repayment period\n1. 7 days\n2. 14 days\n0. Back",
		"loan.select_tenure_7":  "Repayment period: 7 days\n1. Continue\n0. Back",
		"loan.amount_exceeds_tier": "Amount exceeds your tier limit (KES %.0f).\nEnter a lower amount\n0. Back",
		"loan.confirm":        "Apply for %s loan?\nRepay in %d days\n1. Confirm\n2. Cancel",
		"loan.confirm_with_rate": "Apply for %s loan?\nRepay in %d days\nInterest: %s%%\nTier: %s\n1. Confirm\n2. Cancel",
		"loan.confirm_options": "1. Confirm\n2. Cancel",
		"loan.applied_success": "Loan application of %s submitted!\nYou will be notified of the decision.",
		"loan.active_loan":    "You already have an active loan in progress.\nCheck your loan status from the menu.\n0. Back to Menu",
		"loan.apply_failed":   "Loan application failed:\n%s\n0. Back to Menu",

		// Registration
		"register.enter_name":        "Enter your full name\n(First Last)\n0. Back",
		"register.invalid_name":      "Enter first and last name\n0. Back",
		"register.enter_national_id":  "Enter National ID number\n0. Back",
		"register.invalid_national_id": "Invalid ID. Enter 5-12 digits\n0. Back",
		"register.select_role":       "Select your role\n1. Driver\n2. Conductor\n3. Rider\n0. Back",
		"register.enter_pin":         "Create a 4-digit transaction PIN\nThis PIN secures your withdrawals",
		"register.confirm_pin":       "Re-enter your PIN to confirm",
		"register.invalid_pin":       "Invalid PIN. Enter 4-6 digits only",
		"register.pin_mismatch":      "PINs do not match.\nPlease enter your PIN again",
		"register.confirm":           "Register as:\nName: %s\nRole: %s\n1. Confirm\n2. Cancel",
		"register.confirm_options":   "1. Confirm\n2. Cancel",
		"register.success":           "Welcome to CrewPay!\nRegistration complete.\nDial *384*123# to:\n- Check Balance\n- Withdraw\n- View Earnings\n- Access Loans",
		"register.already_registered": "You are already registered!\nDial *384*123# to access\nyour account.",

		// Language
		"language.menu":    "Select Language\n1. English\n2. Kiswahili\n0. Back",
		"language.changed": "Language updated successfully.",

		// Profile
		"profile.menu":               "My Profile\n1. Set PIN\n2. Change PIN\n3. View Profile\n0. Back",
		"profile.set_pin":            "Enter a 4-digit transaction PIN\n0. Back",
		"profile.enter_current_pin":  "Enter your current PIN\n0. Back",
		"profile.enter_new_pin":      "Enter your new PIN (4-6 digits)\n0. Back",
		"profile.pin_set_success":    "Transaction PIN set successfully!\nYou can now make withdrawals.",
		"profile.pin_changed_success": "PIN changed successfully!",
		"profile.no_pin_redirect":    "You don't have a PIN yet.\nPlease create one now.\nEnter a 4-digit PIN",
		"profile.view":               "Your Profile\nPhone: %s\nName: %s",

		// Errors
		"error.generic":             "Something went wrong. Please try again.",
		"error.invalid_input":       "Invalid selection. Try again.",
		"error.not_registered":      "You are not registered.\nDial again and select Register.",
		"error.service_unavailable": "Service temporarily unavailable.\nPlease try again later.",
		"error.session_expired":     "Session expired. Please dial again.",
		"error.rate_limited":        "Too many requests. Please wait and try again.",

		// General
		"goodbye": "Thank you for using CrewPay.\nGoodbye!",
	}

	t.mu.Lock()
	t.messages["en"] = en
	t.mu.Unlock()
}

// --- Swahili Messages ---

func (t *Translator) loadSwahili() {
	sw := map[string]string{
		// Main menu
		"menu.welcome":       "Karibu CrewPay",
		"menu.check_balance": "Angalia Salio",
		"menu.withdraw":      "Toa Pesa",
		"menu.earnings":      "Mapato Yangu",
		"menu.last_payment":  "Malipo ya Mwisho",
		"menu.loan_status":   "Mikopo",
		"menu.my_profile":    "Wasifu Wangu",
		"menu.register":      "Jisajili",
		"menu.language":      "Lugha",
		"menu.exit":          "Toka",
		"menu.back":          "Rudi Menyu",

		// Balance
		"balance.result": "Salio lako: %s",

		// Withdraw
		"withdraw.enter_amount":     "Salio: %s\nIngiza kiasi cha kutoa\n0. Rudi",
		"withdraw.invalid_amount":   "Kiasi batili. Ingiza nambari\n0. Rudi",
		"withdraw.confirm":          "Toa %s?\n1. Thibitisha\n2. Ghairi",
		"withdraw.confirm_options":  "1. Thibitisha\n2. Ghairi",
		"withdraw.enter_pin":        "Ingiza PIN yako ya muamala",
		"withdraw.invalid_pin":      "PIN batili. Ingiza tarakimu 4-6",
		"withdraw.wrong_pin":        "PIN mbaya. Jaribu tena",
		"withdraw.no_pin_set":       "Hujaweka PIN ya muamala.\nNenda Wasifu Wangu (6) kuweka.\n0. Rudi Menyu",
		"withdraw.success":          "Kutoa %s kumeanzishwa.\nRef: %s\nUtapokea M-Pesa hivi karibuni.",
		"withdraw.insufficient_funds": "Salio halitoshi kwa kutoa hii.",

		// Earnings
		"earnings.menu":           "Mapato Yangu\n1. Leo\n2. Wiki Hii\n3. Mwezi Huu\n0. Rudi",
		"earnings.daily_result":   "Mapato ya Leo\nJumla: %s\nKazi: %d",
		"earnings.weekly_result":  "Mapato ya Wiki\nJumla: %s",
		"earnings.monthly_result": "Mapato ya Mwezi\nJumla: %s",

		// Last Payment
		"payment.no_transactions": "Hakuna miamala iliyopatikana.",
		"payment.last_result":     "Muamala wa Mwisho\nAina: %s\nKiasi: %s\nTarehe: %s\nRef: %s",

		// Loans
		"loan.menu":           "Mikopo\n1. Angalia Hali\n2. Omba Mkopo\n3. Maelezo ya Alama\n0. Rudi",
		"loan.no_loans":       "Huna mikopo inayoendelea.",
		"loan.status_result":  "Mkopo: %s\nHali: %s",
		"loan.score_details":  "Alama ya Mkopo: %d (%s)\n\nSababu Kuu:%s\n\nUshauri:%s",
		"loan.low_score":      "Alama yako ya mkopo (%d) ni ndogo sana kuomba mkopo.\nKiwango cha chini kinachohitajika: %d\n0. Rudi kwenye Menyu",
		"loan.tier_info":      "Kiwango Chako: %s\nAlama ya Mkopo: %d\nMkopo wa Juu: KES %.0f\nRiba: %.0f%%\nMuda wa Juu: Siku %d\n\nIngiza kiasi cha mkopo (KES)\n0. Rudi",
		"loan.enter_amount":   "Ingiza kiasi cha mkopo (KES)\n0. Rudi",
		"loan.invalid_amount": "Kiasi batili. Ingiza nambari\n0. Rudi",
		"loan.select_tenure":  "Chagua muda wa kulipa\n1. Siku 7\n2. Siku 14\n3. Siku 30\n0. Rudi",
		"loan.select_tenure_14": "Chagua muda wa kulipa\n1. Siku 7\n2. Siku 14\n0. Rudi",
		"loan.select_tenure_7":  "Muda wa kulipa: Siku 7\n1. Endelea\n0. Rudi",
		"loan.amount_exceeds_tier": "Kiasi kinazidi kikomo chako (KES %.0f).\nIngiza kiasi kidogo\n0. Rudi",
		"loan.confirm":        "Omba mkopo wa %s?\nLipa ndani ya siku %d\n1. Thibitisha\n2. Ghairi",
		"loan.confirm_with_rate": "Omba mkopo wa %s?\nLipa ndani ya siku %d\nRiba: %s%%\nKiwango: %s\n1. Thibitisha\n2. Ghairi",
		"loan.confirm_options": "1. Thibitisha\n2. Ghairi",
		"loan.applied_success": "Maombi ya mkopo wa %s yamewasilishwa!\nUtaarifiwa kuhusu uamuzi.",
		"loan.active_loan":    "Una mkopo unaoendelea tayari.\nAngalia hali ya mkopo wako kwenye menyu.\n0. Rudi kwenye Menyu",
		"loan.apply_failed":   "Maombi ya mkopo yameshindwa:\n%s\n0. Rudi kwenye Menyu",

		// Registration
		"register.enter_name":        "Ingiza jina lako kamili\n(Kwanza Mwisho)\n0. Rudi",
		"register.invalid_name":      "Ingiza jina la kwanza na la mwisho\n0. Rudi",
		"register.enter_national_id":  "Ingiza nambari ya Kitambulisho\n0. Rudi",
		"register.invalid_national_id": "ID batili. Ingiza tarakimu 5-12\n0. Rudi",
		"register.select_role":       "Chagua kazi yako\n1. Dereva\n2. Kondakta\n3. Bodaboda\n0. Rudi",
		"register.enter_pin":         "Unda PIN ya muamala ya tarakimu 4\nPIN hii inalinda fedha zako",
		"register.confirm_pin":       "Ingiza PIN yako tena kuthibitisha",
		"register.invalid_pin":       "PIN batili. Ingiza tarakimu 4-6 tu",
		"register.pin_mismatch":      "PIN hazifanani.\nTafadhali ingiza PIN yako tena",
		"register.confirm":           "Jisajili kama:\nJina: %s\nKazi: %s\n1. Thibitisha\n2. Ghairi",
		"register.confirm_options":   "1. Thibitisha\n2. Ghairi",
		"register.success":           "Karibu CrewPay!\nUsajili umekamilika.\nPiga *384*123# kufikia:\n- Angalia Salio\n- Toa Pesa\n- Mapato\n- Mikopo",
		"register.already_registered": "Tayari umesajiliwa!\nPiga *384*123# kufikia\nakaunti yako.",

		// Language
		"language.menu":    "Chagua Lugha\n1. English\n2. Kiswahili\n0. Rudi",
		"language.changed": "Lugha imebadilishwa.",

		// Profile
		"profile.menu":               "Wasifu Wangu\n1. Weka PIN\n2. Badilisha PIN\n3. Angalia Wasifu\n0. Rudi",
		"profile.set_pin":            "Ingiza PIN ya muamala ya tarakimu 4\n0. Rudi",
		"profile.enter_current_pin":  "Ingiza PIN yako ya sasa\n0. Rudi",
		"profile.enter_new_pin":      "Ingiza PIN yako mpya (tarakimu 4-6)\n0. Rudi",
		"profile.pin_set_success":    "PIN ya muamala imewekwa!\nSasa unaweza kutoa pesa.",
		"profile.pin_changed_success": "PIN imebadilishwa!",
		"profile.no_pin_redirect":    "Huna PIN bado.\nTafadhali unda moja sasa.\nIngiza PIN ya tarakimu 4",
		"profile.view":               "Wasifu Wako\nSimu: %s\nJina: %s",

		// Errors
		"error.generic":             "Kuna tatizo. Jaribu tena.",
		"error.invalid_input":       "Chaguo batili. Jaribu tena.",
		"error.not_registered":      "Hujasajiliwa.\nPiga tena na uchague Jisajili.",
		"error.service_unavailable": "Huduma haipatikani kwa sasa.\nJaribu tena baadaye.",
		"error.session_expired":     "Muda umekwisha. Piga tena.",
		"error.rate_limited":        "Maombi mengi sana. Subiri na ujaribu tena.",

		// General
		"goodbye": "Asante kwa kutumia CrewPay.\nKwaheri!",
	}

	t.mu.Lock()
	t.messages["sw"] = sw
	t.mu.Unlock()
}

// TruncateToLimit truncates a message to the USSD character limit.
// USSD typically allows 160-182 characters per screen.
func TruncateToLimit(msg string, limit int) string {
	if limit <= 0 {
		limit = 160
	}
	if len(msg) <= limit {
		return msg
	}

	// Truncate at the last complete line that fits
	lines := strings.Split(msg[:limit], "\n")
	if len(lines) > 1 {
		return strings.Join(lines[:len(lines)-1], "\n")
	}
	return msg[:limit-3] + "..."
}
