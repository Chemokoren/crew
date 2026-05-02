package ussd

// translations holds all USSD menu text in supported languages.
// Decision D5: Swahili is system default. User pref > Org default > System.
var translations = map[string]map[string]string{
	"sw": {
		// Main menu
		"welcome_menu": "Karibu %s!\n" +
			"1. %s zangu\n" +
			"2. Mapato\n" +
			"3. Pochi\n" +
			"4. Kuingia/Kutoka kazi\n" +
			"5. Wasifu wangu\n" +
			"6. Siku ya malipo",

		// Assignments (G1)
		"assignment_period": `Chagua kipindi cha %s:
1. Leo
2. Wiki hii
3. Mwezi huu`,
		"no_assignments":  "Hakuna %s kwa kipindi hiki.",
		"assignment_list": "%s zangu:\n%s",

		// Check-in/out (G2)
		"no_assignments_today": "Hakuna %s kwa leo.",
		"checkin_select":       "Chagua %s ya kuingia/kutoka:\n%s",
		"confirm_checkin":      "Thibitisha kuingia kazini (%s)?\n1. Ndiyo\n2. Hapana",
		"confirm_checkout":     "Thibitisha kutoka kazini? (Muda: %dh %dm)\n1. Ndiyo\n2. Hapana",
		"checkin_success":      "Umeingia kazini! ✅\nFanya kazi salama.",
		"checkout_success":     "Umetoka kazini! ✅\nMasaa: %.1f\nMalipo yatahesabiwa.",
		"already_complete":     "Kazi hii imekamilika tayari.",
		"action_checkin":       "Ingia",
		"action_checkout":      "Toka",

		// Earnings (G3)
		"earnings_period": `Chagua kipindi cha mapato:
1. Leo
2. Wiki hii
3. Mwezi huu
4. Kipindi cha malipo kilichopita`,
		"earnings_summary": "Mapato (%s - %s):\nJumla: %s\nSiku %d | Malipo %d",

		// Wallet
		"wallet_balance": "Pochi yako:\nSalio: %s\nJumla iliyopatikana: %s\nJumla iliyolipwa: %s",
		"no_wallet":      "Huna pochi bado. Wasiliana na msimamizi wako.",

		// Next payday (G4)
		"next_payday":  "Ratiba: %s (%s)\nSiku ya malipo ijayo:\n%s\n(Siku %d zilizobaki)",
		"no_schedule":  "Hakuna ratiba ya malipo. Wasiliana na msimamizi.",
		"no_org":       "Hujajiunga na shirika lolote bado.",

		// Profile (G5/G7)
		"profile_summary": "Wasifu wako:\nJina: %s\nKitambulisho: %s\nShirika: %s\nNafasi: %s\nSalio: %s",

		// Registration (G6)
		"unregistered_menu":    "Karibu AMY MIS!\nHujasajiliwa.\n1. Jisajili\n2. Ondoka",
		"enter_national_id":    "Ingiza nambari ya kitambulisho:",
		"enter_name":           "Ingiza jina lako kamili (Kwanza Pili):",
		"registration_success": "Umesajiliwa! ✅\nKaribu %s.\nMsimamizi wako atakuweka kazini.",
		"registration_failed":  "Usajili umeshindikana. Jaribu tena baadaye.",
		"registration_partial": "Wasifu umeundwa lakini akaunti haijaundwa. Wasiliana na msimamizi.",
		"already_registered":   "Kitambulisho hiki kimesajiliwa tayari kwa jina %s.",

		// General
		"invalid_option": "Chaguo batili. Jaribu tena.",
		"error_generic":  "Hitilafu imetokea. Jaribu tena.",
		"goodbye":        "Asante kwa kutumia AMY MIS!",
		"cancelled":      "Imeondolewa.",
		"no_crew_profile": "Huna wasifu wa kazi. Wasiliana na msimamizi wako.",
	},
	"en": {
		// Main menu
		"welcome_menu": "Welcome %s!\n" +
			"1. My %ss\n" +
			"2. Earnings\n" +
			"3. Wallet\n" +
			"4. Check In/Out\n" +
			"5. My Profile\n" +
			"6. Next Payday",

		// Assignments (G1)
		"assignment_period": `Select %s period:
1. Today
2. This week
3. This month`,
		"no_assignments":  "No %ss found for this period.",
		"assignment_list": "My %ss:\n%s",

		// Check-in/out (G2)
		"no_assignments_today": "No %ss scheduled for today.",
		"checkin_select":       "Select %s to check in/out:\n%s",
		"confirm_checkin":      "Confirm check-in at %s?\n1. Yes\n2. No",
		"confirm_checkout":     "Confirm check-out? (Duration: %dh %dm)\n1. Yes\n2. No",
		"checkin_success":      "Checked in! ✅\nStay safe at work.",
		"checkout_success":     "Checked out! ✅\nHours: %.1f\nEarnings will be calculated.",
		"already_complete":     "This assignment is already complete.",
		"action_checkin":       "Check In",
		"action_checkout":      "Check Out",

		// Earnings (G3)
		"earnings_period": `Select earnings period:
1. Today
2. This week
3. This month
4. Last pay period`,
		"earnings_summary": "Earnings (%s - %s):\nTotal: %s\nDays %d | Entries %d",

		// Wallet
		"wallet_balance": "Your wallet:\nBalance: %s\nTotal earned: %s\nTotal paid out: %s",
		"no_wallet":      "No wallet yet. Contact your supervisor.",

		// Next payday (G4)
		"next_payday":  "Schedule: %s (%s)\nNext payday:\n%s\n(%d days away)",
		"no_schedule":  "No pay schedule set. Contact your supervisor.",
		"no_org":       "You haven't joined any organization yet.",

		// Profile (G5/G7)
		"profile_summary": "Your profile:\nName: %s\nNational ID: %s\nOrganization: %s\nRole: %s\nBalance: %s",

		// Registration (G6)
		"unregistered_menu":    "Welcome to AMY MIS!\nYou are not registered.\n1. Register\n2. Exit",
		"enter_national_id":    "Enter your National ID number:",
		"enter_name":           "Enter your full name (First Last):",
		"registration_success": "Registered! ✅\nWelcome %s.\nYour supervisor will assign you work.",
		"registration_failed":  "Registration failed. Try again later.",
		"registration_partial": "Profile created but account linking failed. Contact your supervisor.",
		"already_registered":   "This ID is already registered under %s.",

		// General
		"invalid_option": "Invalid option. Try again.",
		"error_generic":  "An error occurred. Try again.",
		"goodbye":        "Thank you for using AMY MIS!",
		"cancelled":      "Cancelled.",
		"no_crew_profile": "No crew profile found. Contact your supervisor.",
	},
}

// t returns a translated string for the given language and key.
// Falls back to Swahili if key not found in the requested language.
func t(lang, key string) string {
	if msgs, ok := translations[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	// Fallback to Swahili
	if msgs, ok := translations["sw"]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	return key
}
