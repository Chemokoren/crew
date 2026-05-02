package ussd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"log/slog"
)

// SessionHandler processes Africa's Talking USSD callbacks.
// AT sends: sessionId, phoneNumber, serviceCode, text (accumulated pipe-delimited input).
// We return: CON <menu> (continue) or END <message> (terminate).
type SessionHandler struct {
	userRepo       repository.UserRepository
	crewRepo       repository.CrewRepository
	assignmentRepo repository.AssignmentRepository
	earningRepo    repository.EarningRepository
	orgRepo        repository.OrganizationRepository
	jobTypeRepo    repository.TenantJobTypeRepository
	scheduleRepo   repository.PayScheduleRepository
	membershipRepo repository.MembershipRepository
	walletRepo     repository.WalletRepository
	logger         *slog.Logger
}

// NewSessionHandler creates a new USSD session handler.
func NewSessionHandler(
	userRepo repository.UserRepository,
	crewRepo repository.CrewRepository,
	assignmentRepo repository.AssignmentRepository,
	earningRepo repository.EarningRepository,
	orgRepo repository.OrganizationRepository,
	jobTypeRepo repository.TenantJobTypeRepository,
	scheduleRepo repository.PayScheduleRepository,
	membershipRepo repository.MembershipRepository,
	walletRepo repository.WalletRepository,
	logger *slog.Logger,
) *SessionHandler {
	return &SessionHandler{
		userRepo:       userRepo,
		crewRepo:       crewRepo,
		assignmentRepo: assignmentRepo,
		earningRepo:    earningRepo,
		orgRepo:        orgRepo,
		jobTypeRepo:    jobTypeRepo,
		scheduleRepo:   scheduleRepo,
		membershipRepo: membershipRepo,
		walletRepo:     walletRepo,
		logger:         logger,
	}
}

// USSDRequest represents an incoming USSD callback from Africa's Talking.
type USSDRequest struct {
	SessionID   string `form:"sessionId" json:"sessionId"`
	PhoneNumber string `form:"phoneNumber" json:"phoneNumber"`
	ServiceCode string `form:"serviceCode" json:"serviceCode"`
	Text        string `form:"text" json:"text"`
}

// sessionContext bundles the resolved user, crew, org, and labels for a session.
type sessionContext struct {
	user   *models.User
	crew   *models.CrewMember
	org    *models.SACCO // Organization (aliased as SACCO in models)
	lang   string
	labels IndustryLabels
}

// HandleSession processes the USSD session and returns the response text.
// The text field contains the full path of user inputs, separated by *.
func (h *SessionHandler) HandleSession(ctx context.Context, req USSDRequest) string {
	phone := normalizePhone(req.PhoneNumber)
	parts := parsePath(req.Text)
	level := len(parts)

	h.logger.Info("ussd session",
		slog.String("session_id", req.SessionID),
		slog.String("phone", phone),
		slog.Int("level", level),
		slog.String("text", req.Text),
	)

	// Step 1: Find user by phone
	user, err := h.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		// Not registered — offer registration
		return h.handleUnregistered(ctx, parts, phone)
	}

	// Step 2: Load crew member profile
	var crew *models.CrewMember
	if user.CrewMemberID != nil {
		crew, err = h.crewRepo.GetByID(ctx, *user.CrewMemberID)
		if err != nil {
			crew = nil
		}
	}

	// Step 3: Resolve language (D5: user > org > system)
	sc := h.buildSessionContext(ctx, user, crew)

	if level == 0 {
		return h.mainMenu(sc)
	}

	switch parts[0] {
	case "1": // My Assignments/Shifts/Jobs/Visits
		return h.handleAssignments(ctx, sc, parts[1:])
	case "2": // Earnings
		return h.handleEarnings(ctx, sc, parts[1:])
	case "3": // Wallet
		return h.handleWallet(ctx, sc)
	case "4": // Check In/Out
		return h.handleCheckInOut(ctx, sc, parts[1:])
	case "5": // My Profile
		return h.handleProfile(ctx, sc)
	case "6": // Next Payday
		return h.handleNextPayday(ctx, sc)
	default:
		return end(t(sc.lang, "invalid_option"))
	}
}

// buildSessionContext resolves language, org, and industry labels.
func (h *SessionHandler) buildSessionContext(ctx context.Context, user *models.User, crew *models.CrewMember) sessionContext {
	sc := sessionContext{
		user: user,
		crew: crew,
		lang: models.SystemDefaultLanguage,
		labels: IndustryLabels{
			Assignment: "Assignment", WorkSite: "Location",
			Worker: "Worker", Organization: "Organization", Vehicle: "Vehicle",
		},
	}

	// Resolve org
	if user.OrganizationID != nil {
		org, err := h.orgRepo.GetByID(ctx, *user.OrganizationID)
		if err == nil {
			sc.org = org
		}
	}

	// Resolve language: user pref > org default > system
	orgDefault := ""
	if sc.org != nil {
		orgDefault = sc.org.DefaultLanguage
	}
	sc.lang = models.ResolveLanguage(user.PreferredLanguage, orgDefault)

	// Resolve industry labels from org template
	if sc.org != nil {
		tmpl := models.GetIndustryTemplate(sc.org.IndustryType)
		if tmpl.UILabels != nil {
			if v, ok := tmpl.UILabels["assignment"]; ok {
				sc.labels.Assignment = v
			}
			if v, ok := tmpl.UILabels["work_site"]; ok {
				sc.labels.WorkSite = v
			}
			if v, ok := tmpl.UILabels["worker"]; ok {
				sc.labels.Worker = v
			}
			if v, ok := tmpl.UILabels["organization"]; ok {
				sc.labels.Organization = v
			}
			if v, ok := tmpl.UILabels["vehicle"]; ok {
				sc.labels.Vehicle = v
			}
		}
	}

	return sc
}

// --- Main Menu ---

func (h *SessionHandler) mainMenu(sc sessionContext) string {
	name := ""
	if sc.crew != nil {
		name = sc.crew.FirstName
	}
	return con(fmt.Sprintf(t(sc.lang, "welcome_menu"), name, sc.labels.Assignment))
}

// --- G1: Industry-Aware Assignments ---

func (h *SessionHandler) handleAssignments(ctx context.Context, sc sessionContext, parts []string) string {
	if sc.crew == nil {
		return end(t(sc.lang, "no_crew_profile"))
	}

	if len(parts) == 0 {
		return con(fmt.Sprintf(t(sc.lang, "assignment_period"), sc.labels.Assignment))
	}

	dateFrom, dateTo := h.resolvePeriod(parts[0])
	if dateFrom.IsZero() {
		return end(t(sc.lang, "invalid_option"))
	}

	assignments, _, _ := h.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &sc.crew.ID,
		DateFrom:     &dateFrom,
		DateTo:       &dateTo,
	}, 1, 5)

	if len(assignments) == 0 {
		return end(fmt.Sprintf(t(sc.lang, "no_assignments"), sc.labels.Assignment))
	}

	var lines []string
	for i, a := range assignments {
		status := "⏳"
		if a.Status == models.AssignmentCompleted {
			status = "✅"
		}
		if a.Status == models.AssignmentActive {
			status = "🔄"
		}

		location := h.resolveLocation(a, sc)
		lines = append(lines, fmt.Sprintf("%d. %s %s %s (%s)",
			i+1, status, a.ShiftDate.Format("02/01"),
			location, a.WorkType))
	}

	return end(fmt.Sprintf(t(sc.lang, "assignment_list"),
		sc.labels.Assignment, strings.Join(lines, "\n")))
}

// --- G2: Check-In/Check-Out ---

func (h *SessionHandler) handleCheckInOut(ctx context.Context, sc sessionContext, parts []string) string {
	if sc.crew == nil {
		return end(t(sc.lang, "no_crew_profile"))
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.Add(24 * time.Hour)

	assignments, _, _ := h.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &sc.crew.ID,
		DateFrom:     &today,
		DateTo:       &tomorrow,
	}, 1, 5)

	if len(assignments) == 0 {
		return end(fmt.Sprintf(t(sc.lang, "no_assignments_today"), sc.labels.Assignment))
	}

	if len(parts) == 0 {
		// Show today's assignments for selection
		var lines []string
		for i, a := range assignments {
			action := t(sc.lang, "action_checkin")
			if a.CheckInAt != nil && a.CheckOutAt == nil {
				action = t(sc.lang, "action_checkout")
			} else if a.CheckOutAt != nil {
				action = "✅"
			}
			location := h.resolveLocation(a, sc)
			lines = append(lines, fmt.Sprintf("%d. %s — %s", i+1, location, action))
		}
		return con(fmt.Sprintf(t(sc.lang, "checkin_select"),
			sc.labels.Assignment, strings.Join(lines, "\n")))
	}

	// User selected an assignment
	idx := 0
	fmt.Sscanf(parts[0], "%d", &idx)
	if idx < 1 || idx > len(assignments) {
		return end(t(sc.lang, "invalid_option"))
	}

	assignment := assignments[idx-1]

	if len(parts) == 1 {
		// Confirm action
		location := h.resolveLocation(assignment, sc)
		if assignment.CheckInAt == nil {
			return con(fmt.Sprintf(t(sc.lang, "confirm_checkin"), location))
		} else if assignment.CheckOutAt == nil {
			elapsed := now.Sub(*assignment.CheckInAt)
			hours := int(elapsed.Hours())
			mins := int(elapsed.Minutes()) % 60
			return con(fmt.Sprintf(t(sc.lang, "confirm_checkout"), hours, mins))
		}
		return end(t(sc.lang, "already_complete"))
	}

	if parts[1] == "1" {
		// Execute check-in or check-out
		if assignment.CheckInAt == nil {
			assignment.CheckInAt = &now
			assignment.Status = models.AssignmentActive
		} else if assignment.CheckOutAt == nil {
			assignment.CheckOutAt = &now
			assignment.Status = models.AssignmentCompleted
			if assignment.CheckInAt != nil {
				duration := now.Sub(*assignment.CheckInAt)
				hours := duration.Hours()
				assignment.HoursWorked = &hours
			}
		}
		if err := h.assignmentRepo.Update(ctx, &assignment); err != nil {
			h.logger.Error("ussd: check-in/out failed", slog.Any("err", err))
			return end(t(sc.lang, "error_generic"))
		}

		if assignment.CheckOutAt != nil {
			return end(fmt.Sprintf(t(sc.lang, "checkout_success"), *assignment.HoursWorked))
		}
		return end(t(sc.lang, "checkin_success"))
	}

	return end(t(sc.lang, "cancelled"))
}

// --- G3: Earnings by Period ---

func (h *SessionHandler) handleEarnings(ctx context.Context, sc sessionContext, parts []string) string {
	if sc.crew == nil {
		return end(t(sc.lang, "no_crew_profile"))
	}

	if len(parts) == 0 {
		return con(t(sc.lang, "earnings_period"))
	}

	dateFrom, dateTo := h.resolvePeriod(parts[0])
	if parts[0] == "4" {
		// Last pay period — resolve from actual pay schedule
		dateFrom, dateTo = h.resolveLastPayPeriod(ctx, sc)
	}
	if dateFrom.IsZero() {
		return end(t(sc.lang, "invalid_option"))
	}

	earnings, _, _ := h.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &sc.crew.ID,
		DateFrom:     &dateFrom,
		DateTo:       &dateTo,
	}, 1, 1000)

	var total int64
	days := make(map[string]bool)
	for _, e := range earnings {
		total += e.AmountCents
		days[e.EarnedAt.Format("2006-01-02")] = true
	}

	return end(fmt.Sprintf(t(sc.lang, "earnings_summary"),
		dateFrom.Format("02/01"), dateTo.Format("02/01"),
		formatKES(total), len(days), len(earnings),
	))
}

// --- G3: Wallet Balance ---

func (h *SessionHandler) handleWallet(ctx context.Context, sc sessionContext) string {
	if sc.crew == nil {
		return end(t(sc.lang, "no_crew_profile"))
	}

	wallet, err := h.walletRepo.GetByCrewMemberID(ctx, sc.crew.ID)
	if err != nil {
		return end(t(sc.lang, "no_wallet"))
	}

	return end(fmt.Sprintf(t(sc.lang, "wallet_balance"),
		formatKES(wallet.BalanceCents),
		formatKES(wallet.TotalCreditedCents),
		formatKES(wallet.TotalDebitedCents),
	))
}

// --- G4: Next Payday ---

func (h *SessionHandler) handleNextPayday(ctx context.Context, sc sessionContext) string {
	if sc.org == nil {
		return end(t(sc.lang, "no_org"))
	}

	schedules, err := h.scheduleRepo.ListByOrganization(ctx, sc.org.ID)
	if err != nil || len(schedules) == 0 {
		return end(t(sc.lang, "no_schedule"))
	}

	// Use default schedule (or first)
	sched := schedules[0]
	for _, s := range schedules {
		if s.IsDefault {
			sched = s
			break
		}
	}

	nextPayday := calculateNextPayday(sched, time.Now())

	return end(fmt.Sprintf(t(sc.lang, "next_payday"),
		sched.Name,
		string(sched.Frequency),
		nextPayday.Format("Monday, 02 Jan 2006"),
		daysUntil(nextPayday),
	))
}

// --- G5: Profile & G7: Financial Summary ---

func (h *SessionHandler) handleProfile(ctx context.Context, sc sessionContext) string {
	if sc.crew == nil {
		return end(t(sc.lang, "no_crew_profile"))
	}

	balanceStr := "N/A"
	wallet, _ := h.walletRepo.GetByCrewMemberID(ctx, sc.crew.ID)
	if wallet != nil {
		balanceStr = formatKES(wallet.BalanceCents)
	}

	orgName := "—"
	if sc.org != nil {
		orgName = sc.org.Name
	}

	return end(fmt.Sprintf(t(sc.lang, "profile_summary"),
		sc.crew.FirstName+" "+sc.crew.LastName,
		sc.crew.NationalID,
		orgName,
		string(sc.crew.Role),
		balanceStr,
	))
}

// --- G6: Unregistered User — Registration Flow ---

func (h *SessionHandler) handleUnregistered(ctx context.Context, parts []string, phone string) string {
	if len(parts) == 0 {
		return con(t("sw", "unregistered_menu"))
	}

	if parts[0] == "1" {
		// Registration flow: 1 → NationalID → FullName
		if len(parts) < 2 {
			return con(t("sw", "enter_national_id"))
		}
		if len(parts) < 3 {
			return con(t("sw", "enter_name"))
		}

		nationalID := parts[1]
		names := strings.SplitN(parts[2], " ", 2)
		firstName := names[0]
		lastName := ""
		if len(names) > 1 {
			lastName = names[1]
		}

		// Check if national ID already exists
		existing, err := h.crewRepo.GetByNationalID(ctx, nationalID)
		if err == nil && existing != nil {
			return end(fmt.Sprintf(t("sw", "already_registered"), existing.FirstName))
		}

		// Generate crew ID
		crewID, err := h.crewRepo.NextCrewID(ctx)
		if err != nil {
			h.logger.Error("ussd: failed to generate crew ID", slog.Any("err", err))
			return end(t("sw", "registration_failed"))
		}

		// Create crew member
		newCrew := &models.CrewMember{
			CrewID:     crewID,
			NationalID: nationalID,
			FirstName:  firstName,
			LastName:   lastName,
			Role:       models.RoleOther,
			IsActive:   true,
		}
		if err := h.crewRepo.Create(ctx, newCrew); err != nil {
			h.logger.Error("ussd: registration failed", slog.Any("err", err))
			return end(t("sw", "registration_failed"))
		}

		// Create user record linked to crew member
		newUser := &models.User{
			Phone:             phone,
			PasswordHash:      "", // PIN-only auth for USSD
			SystemRole:        "CREW",
			CrewMemberID:      &newCrew.ID,
			PreferredLanguage: "sw",
			IsActive:          true,
		}
		if err := h.userRepo.Create(ctx, newUser); err != nil {
			h.logger.Error("ussd: user creation failed", slog.Any("err", err))
			// Best effort — crew profile created, user linking failed
			return end(t("sw", "registration_partial"))
		}

		return end(fmt.Sprintf(t("sw", "registration_success"), firstName))
	}

	return end(t("sw", "goodbye"))
}

// --- Shared Helpers ---

// IndustryLabels holds UI labels resolved from the org's industry template (AD-9/G1).
type IndustryLabels struct {
	Assignment   string
	WorkSite     string
	Worker       string
	Organization string
	Vehicle      string
}

// resolveLocation returns the best location string for an assignment (G5: work site for non-transport).
func (h *SessionHandler) resolveLocation(a models.Assignment, sc sessionContext) string {
	// G5: Prefer work_site for non-transport
	if a.WorkSite != "" {
		return a.WorkSite
	}
	if a.VehicleID != nil && a.Vehicle.RegistrationNo != "" {
		return a.Vehicle.RegistrationNo
	}
	if a.ProjectRef != "" {
		return "Proj: " + a.ProjectRef
	}
	return sc.labels.WorkSite
}

func (h *SessionHandler) resolvePeriod(choice string) (time.Time, time.Time) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch choice {
	case "1": // Today
		return today, today.Add(24 * time.Hour)
	case "2": // This week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1))
		return start, start.AddDate(0, 0, 7)
	case "3": // This month
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		return time.Time{}, time.Time{}
	}
}

func (h *SessionHandler) resolveLastPayPeriod(ctx context.Context, sc sessionContext) (time.Time, time.Time) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if sc.org == nil {
		return today.AddDate(0, -1, 0), today
	}

	schedules, _ := h.scheduleRepo.ListByOrganization(ctx, sc.org.ID)
	if len(schedules) > 0 {
		return resolvePeriodWindow(schedules[0], now)
	}

	return today.AddDate(0, -1, 0), today
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		phone = "254" + phone[1:]
	}
	return phone
}

func parsePath(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	return strings.Split(text, "*")
}

func con(msg string) string { return "CON " + msg }
func end(msg string) string { return "END " + msg }

func formatKES(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("KES %d.%02d", whole, frac)
}

func daysUntil(target time.Time) int {
	now := time.Now()
	diff := target.Sub(now)
	days := int(diff.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func calculateNextPayday(sched models.PaySchedule, now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch sched.Frequency {
	case models.PayDaily:
		return today.Add(24 * time.Hour)

	case models.PayWeekly:
		payDay := 5 // Friday default
		if sched.PayDay != nil {
			payDay = *sched.PayDay
		}
		current := int(now.Weekday())
		if current == 0 {
			current = 7
		}
		diff := payDay - current
		if diff <= 0 {
			diff += 7
		}
		return today.AddDate(0, 0, diff)

	case models.PayBiWeekly:
		payDay := 5
		if sched.PayDay != nil {
			payDay = *sched.PayDay
		}
		current := int(now.Weekday())
		if current == 0 {
			current = 7
		}
		diff := payDay - current
		if diff <= 0 {
			diff += 14
		}
		return today.AddDate(0, 0, diff)

	case models.PayMonthly:
		payDay := 28
		if sched.PayDay != nil {
			payDay = *sched.PayDay
		}
		target := time.Date(now.Year(), now.Month(), payDay, 0, 0, 0, 0, time.UTC)
		if !target.After(today) {
			target = target.AddDate(0, 1, 0)
		}
		return target

	default:
		return today.Add(24 * time.Hour)
	}
}

func resolvePeriodWindow(sched models.PaySchedule, now time.Time) (time.Time, time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch sched.Frequency {
	case models.PayDaily:
		return today.Add(-24 * time.Hour), today
	case models.PayWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1 + 7))
		return start, start.AddDate(0, 0, 7)
	case models.PayBiWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1 + 14))
		return start, start.AddDate(0, 0, 14)
	case models.PayMonthly:
		lastMonth := now.AddDate(0, -1, 0)
		start := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		return today.Add(-24 * time.Hour), today
	}
}
