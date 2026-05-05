package models

// IndustryTemplate provides pre-configured settings for an industry type.
// Implements Decision D4 (construction first) and D6 (layered config abstraction).
type IndustryTemplate struct {
	IndustryType    IndustryType     `json:"industry_type"`
	OrgType         OrganizationType `json:"organization_type"`
	DisplayLabel    string           `json:"display_label"`
	AssignmentTypes []string         `json:"assignment_types"`
	EarningModels   []string         `json:"earning_models"`
	Frequencies     []string         `json:"payment_frequencies"`
	StatutoryBodies []string         `json:"statutory_bodies"`
	DefaultJobTypes []DefaultJobType `json:"default_job_types"`
	UILabels        map[string]string `json:"ui_labels"`
}

// DefaultJobType is a template job type seeded with an industry template.
type DefaultJobType struct {
	Code        string          `json:"code"`
	DisplayName string          `json:"display_name"`
	Category    JobTypeCategory `json:"category"`
}

// GetIndustryTemplate returns the pre-configured template for an industry type.
func GetIndustryTemplate(industry IndustryType) IndustryTemplate {
	templates := map[IndustryType]IndustryTemplate{
		IndustryTransport: {
			IndustryType:    IndustryTransport,
			OrgType:         OrgTypeSacco,
			DisplayLabel:    "Transport SACCO",
			AssignmentTypes: []string{"SHIFT"},
			EarningModels:   []string{"FIXED", "COMMISSION", "HYBRID"},
			Frequencies:     []string{"DAILY", "WEEKLY"},
			StatutoryBodies: []string{"SHA", "NSSF", "HousingLevy"},
			DefaultJobTypes: []DefaultJobType{
				{Code: "DRIVER", DisplayName: "Driver", Category: JobCategoryPrimary},
				{Code: "CONDUCTOR", DisplayName: "Conductor", Category: JobCategoryPrimary},
				{Code: "RIDER", DisplayName: "Boda Rider", Category: JobCategoryPrimary},
				{Code: "BOOKING_AGENT", DisplayName: "Booking Agent", Category: JobCategoryFacilitator},
				{Code: "DISPATCHER", DisplayName: "Dispatcher", Category: JobCategoryFacilitator},
				{Code: "SUPERVISOR", DisplayName: "Fleet Supervisor", Category: JobCategorySupervisor},
				{Code: "OFFICE_ADMIN", DisplayName: "Office Administrator", Category: JobCategorySupport},
			},
			UILabels: map[string]string{
				"assignment": "Shift", "work_site": "Route", "worker": "Crew Member",
				"organization": "SACCO", "vehicle": "Vehicle",
			},
		},
		IndustryConstruction: {
			IndustryType:    IndustryConstruction,
			OrgType:         OrgTypeConstructionFirm,
			DisplayLabel:    "Construction Firm",
			AssignmentTypes: []string{"DAILY", "HOURLY", "PROJECT"},
			EarningModels:   []string{"DAILY_RATE", "HOURLY", "PER_TASK", "FIXED"},
			Frequencies:     []string{"DAILY", "WEEKLY", "BI_WEEKLY"},
			StatutoryBodies: []string{}, // Casual labor — opt-in (D7)
			DefaultJobTypes: []DefaultJobType{
				{Code: "MASON", DisplayName: "Mason", Category: JobCategoryPrimary},
				{Code: "CARPENTER", DisplayName: "Carpenter", Category: JobCategoryPrimary},
				{Code: "PLUMBER", DisplayName: "Plumber", Category: JobCategoryPrimary},
				{Code: "ELECTRICIAN", DisplayName: "Electrician", Category: JobCategoryPrimary},
				{Code: "LABORER", DisplayName: "General Laborer", Category: JobCategoryPrimary},
				{Code: "FOREMAN", DisplayName: "Foreman", Category: JobCategorySupervisor},
				{Code: "SITE_MANAGER", DisplayName: "Site Manager", Category: JobCategorySupervisor},
				{Code: "SAFETY_OFFICER", DisplayName: "Safety Officer", Category: JobCategorySupport},
			},
			UILabels: map[string]string{
				"assignment": "Job", "work_site": "Site", "worker": "Worker",
				"organization": "Contractor", "vehicle": "Equipment",
			},
		},
		IndustryHealth: {
			IndustryType:    IndustryHealth,
			OrgType:         OrgTypeHealthNGO,
			DisplayLabel:    "Health Organization",
			AssignmentTypes: []string{"DAILY", "HOURLY", "TASK"},
			EarningModels:   []string{"DAILY_RATE", "PER_TASK", "SALARY"},
			Frequencies:     []string{"MONTHLY", "BI_WEEKLY"},
			StatutoryBodies: []string{"SHA"},
			DefaultJobTypes: []DefaultJobType{
				{Code: "CHV", DisplayName: "Community Health Volunteer", Category: JobCategoryPrimary},
				{Code: "CHP", DisplayName: "Community Health Promoter", Category: JobCategoryPrimary},
				{Code: "NURSE", DisplayName: "Nurse", Category: JobCategoryPrimary},
				{Code: "COORDINATOR", DisplayName: "Coordinator", Category: JobCategorySupervisor},
				{Code: "DATA_CLERK", DisplayName: "Data Clerk", Category: JobCategorySupport},
			},
			UILabels: map[string]string{
				"assignment": "Visit", "work_site": "Coverage Area", "worker": "Health Worker",
				"organization": "Health Facility", "vehicle": "Transport",
			},
		},
		IndustryLogistics: {
			IndustryType:    IndustryLogistics,
			OrgType:         OrgTypeLogisticsCompany,
			DisplayLabel:    "Logistics Company",
			AssignmentTypes: []string{"TASK", "SHIFT", "DAILY"},
			EarningModels:   []string{"PER_TASK", "FIXED", "COMMISSION"},
			Frequencies:     []string{"DAILY", "WEEKLY"},
			StatutoryBodies: []string{},
			DefaultJobTypes: []DefaultJobType{
				{Code: "RIDER", DisplayName: "Delivery Rider", Category: JobCategoryPrimary},
				{Code: "DRIVER", DisplayName: "Driver", Category: JobCategoryPrimary},
				{Code: "LOADER", DisplayName: "Loader", Category: JobCategoryPrimary},
				{Code: "DISPATCHER", DisplayName: "Dispatcher", Category: JobCategoryFacilitator},
				{Code: "WAREHOUSE_MGR", DisplayName: "Warehouse Manager", Category: JobCategorySupervisor},
			},
			UILabels: map[string]string{
				"assignment": "Delivery", "work_site": "Warehouse", "worker": "Worker",
				"organization": "Company", "vehicle": "Vehicle",
			},
		},
		IndustryAgriculture: {
			IndustryType:    IndustryAgriculture,
			OrgType:         OrgTypeAgricultureCoop,
			DisplayLabel:    "Agricultural Cooperative",
			AssignmentTypes: []string{"DAILY", "HOURLY", "PROJECT"},
			EarningModels:   []string{"DAILY_RATE", "HOURLY", "PER_PIECE"},
			Frequencies:     []string{"DAILY", "WEEKLY"},
			StatutoryBodies: []string{},
			DefaultJobTypes: []DefaultJobType{
				{Code: "PICKER", DisplayName: "Picker / Harvester", Category: JobCategoryPrimary},
				{Code: "FIELD_WORKER", DisplayName: "Field Worker", Category: JobCategoryPrimary},
				{Code: "SORTER", DisplayName: "Sorter / Grader", Category: JobCategoryPrimary},
				{Code: "TEAM_LEAD", DisplayName: "Team Leader", Category: JobCategorySupervisor},
				{Code: "WEIGHBRIDGE", DisplayName: "Weighbridge Operator", Category: JobCategorySupport},
			},
			UILabels: map[string]string{
				"assignment": "Task", "work_site": "Farm Block", "worker": "Worker",
				"organization": "Cooperative", "vehicle": "Tractor",
			},
		},
		IndustryHospitality: {
			IndustryType:    IndustryHospitality,
			OrgType:         OrgTypeHospitalityGroup,
			DisplayLabel:    "Hospitality Group",
			AssignmentTypes: []string{"SHIFT", "DAILY"},
			EarningModels:   []string{"DAILY_RATE", "HOURLY", "SALARY"},
			Frequencies:     []string{"WEEKLY", "BI_WEEKLY", "MONTHLY"},
			StatutoryBodies: []string{"SHA", "NSSF"},
			DefaultJobTypes: []DefaultJobType{
				{Code: "WAITER", DisplayName: "Waiter / Waitress", Category: JobCategoryPrimary},
				{Code: "COOK", DisplayName: "Cook", Category: JobCategoryPrimary},
				{Code: "HOUSEKEEPER", DisplayName: "Housekeeper", Category: JobCategoryPrimary},
				{Code: "RECEPTIONIST", DisplayName: "Receptionist", Category: JobCategorySupport},
				{Code: "SHIFT_LEAD", DisplayName: "Shift Lead", Category: JobCategorySupervisor},
			},
			UILabels: map[string]string{
				"assignment": "Shift", "work_site": "Location", "worker": "Staff",
				"organization": "Establishment", "vehicle": "Transport",
			},
		},
	}

	if t, ok := templates[industry]; ok {
		return t
	}
	// General fallback
	return IndustryTemplate{
		IndustryType:    IndustryGeneral,
		OrgType:         OrgTypeGeneral,
		DisplayLabel:    "Organization",
		AssignmentTypes: []string{"DAILY", "SHIFT"},
		EarningModels:   []string{"FIXED", "DAILY_RATE"},
		Frequencies:     []string{"DAILY", "WEEKLY"},
		StatutoryBodies: []string{},
		DefaultJobTypes: []DefaultJobType{
			{Code: "WORKER", DisplayName: "Worker", Category: JobCategoryPrimary},
			{Code: "SUPERVISOR", DisplayName: "Supervisor", Category: JobCategorySupervisor},
		},
		UILabels: map[string]string{
			"assignment": "Assignment", "work_site": "Location", "worker": "Worker",
			"organization": "Organization", "vehicle": "Vehicle",
		},
	}
}
