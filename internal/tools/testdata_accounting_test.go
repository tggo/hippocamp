package tools

func accountingProject() testProject {
	const base = "https://hippocamp.dev/project/accounting/"

	topicURI := func(slug string) string { return base + "topic/" + slug }
	entityURI := func(slug string) string { return base + "entity/" + slug }
	noteURI := func(slug string) string { return base + "note/" + slug }
	decisionURI := func(slug string) string { return base + "decision/" + slug }
	questionURI := func(slug string) string { return base + "question/" + slug }
	sourceURI := func(slug string) string { return base + "source/" + slug }
	tagURI := func(slug string) string { return base + "tag/" + slug }

	var triples []testTriple
	add := func(tt []testTriple) { triples = append(triples, tt...) }

	// Topics
	add(topic(topicURI("revenue"), "Revenue", "Income tracking, invoicing, and revenue recognition"))
	add(topic(topicURI("expenses"), "Expenses", "Business expense tracking and categorization"))
	add(topic(topicURI("tax-planning"), "Tax Planning", "Federal and state tax strategy and compliance"))
	add(topic(topicURI("payroll"), "Payroll", "Employee and contractor payroll processing"))
	add(topic(topicURI("accounts-receivable"), "Accounts Receivable", "Outstanding invoices and collections"))
	add(topic(topicURI("depreciation"), "Depreciation", "Asset depreciation schedules and methods"))
	add(topic(topicURI("monthly-close"), "Monthly Close", "Month-end close procedures and reconciliation"))

	// Tags
	add(tag(tagURI("urgent"), "urgent"))
	add(tag(tagURI("compliance"), "compliance"))

	// Entities
	add(entity(entityURI("riverside-cafe"), "Riverside Cafe LLC", "Primary business entity, cafe and catering operation in Portland, Oregon"))
	triples = append(triples,
		tripleHasTopic(entityURI("riverside-cafe"), topicURI("revenue")),
	)

	add(entity(entityURI("rachel-green"), "Rachel Green CPA", "External accountant at Green & Associates, handles annual tax filings and financial review"))
	triples = append(triples,
		tripleHasTopic(entityURI("rachel-green"), topicURI("tax-planning")),
	)

	add(entity(entityURI("maria-santos"), "Maria Santos", "Part-time bookkeeper, 20 hours per week, manages day-to-day transaction entry and reconciliation"))
	triples = append(triples,
		tripleHasTopic(entityURI("maria-santos"), topicURI("monthly-close")),
	)

	add(entity(entityURI("morning-brew-catering"), "Morning Brew Catering", "Regular catering client, $8K monthly contract for corporate breakfast service"))
	triples = append(triples,
		tripleHasTopic(entityURI("morning-brew-catering"), topicURI("accounts-receivable")),
	)

	add(entity(entityURI("riverside-events"), "Riverside Events", "Event hosting client, $12K monthly revenue from private dining and event space rental"))
	triples = append(triples,
		tripleHasTopic(entityURI("riverside-events"), topicURI("accounts-receivable")),
	)

	add(entity(entityURI("oregon-dor"), "Oregon Department of Revenue", "State tax authority for Oregon business tax filings and quarterly estimated payments"))
	triples = append(triples,
		tripleHasTopic(entityURI("oregon-dor"), topicURI("tax-planning")),
		tripleTag(entityURI("oregon-dor"), tagURI("compliance")),
	)

	add(entity(entityURI("quickbooks-online"), "QuickBooks Online", "Primary accounting software, Plus plan, connected to bank feeds and payroll"))
	triples = append(triples,
		tripleHasTopic(entityURI("quickbooks-online"), topicURI("monthly-close")),
	)

	// Notes
	add(note(noteURI("chart-of-accounts"), "Chart of Accounts Structure",
		"Chart of accounts: Revenue accounts 4000-4999 (4010 Dine-in, 4020 Catering, 4030 Events). COGS 5000-5999 (5010 Food, 5020 Beverage). Operating Expenses 6000-6999 (6010 Rent, 6020 Utilities, 6030 Labor)."))
	triples = append(triples,
		tripleHasTopic(noteURI("chart-of-accounts"), topicURI("revenue")),
	)

	add(note(noteURI("q1-summary"), "Q1 Financial Summary",
		"Q1 results: $127K total revenue across all channels. 340 invoices processed. Accounts receivable: $15K overdue beyond 30 days, primarily from two catering clients. Net profit margin 11.2%."))
	triples = append(triples,
		tripleHasTopic(noteURI("q1-summary"), topicURI("revenue")),
		tripleHasTopic(noteURI("q1-summary"), topicURI("accounts-receivable")),
	)

	add(note(noteURI("month-end-checklist"), "Month-End Close Checklist",
		"Monthly close checklist: 1) Bank reconciliation for all accounts. 2) Credit card statement review and categorization. 3) Record depreciation entries for fixed assets. 4) Accrue payroll for partial pay periods. 5) Review AR aging report. 6) Generate P&L and balance sheet."))
	triples = append(triples,
		tripleHasTopic(noteURI("month-end-checklist"), topicURI("monthly-close")),
	)

	add(note(noteURI("section-179"), "Section 179 Deduction — Espresso Machine",
		"Section 179 deduction for the $18K commercial espresso machine purchased in January. Qualifies as business equipment. Full deduction in year of purchase rather than depreciating over 7 years. Need to file Form 4562."))
	triples = append(triples,
		tripleHasTopic(noteURI("section-179"), topicURI("depreciation")),
		tripleHasTopic(noteURI("section-179"), topicURI("tax-planning")),
	)

	// Decisions
	add(decision(decisionURI("accrual-accounting"), "Switched from Cash to Accrual Accounting",
		"Switched from cash basis to accrual accounting as required by SBA loan application. Accrual provides more accurate picture of financial health and is required for businesses seeking loans over $250K."))
	triples = append(triples,
		tripleHasTopic(decisionURI("accrual-accounting"), topicURI("revenue")),
		tripleTag(decisionURI("accrual-accounting"), tagURI("compliance")),
	)

	add(decision(decisionURI("hire-bookkeeper"), "Hired Part-Time Bookkeeper",
		"Hired Maria Santos as part-time bookkeeper (20 hrs/week) to improve transaction accuracy. Previous error rate was 4.2%, now below 0.5%. Cost justified by reduced CPA review hours."))
	triples = append(triples,
		tripleHasTopic(decisionURI("hire-bookkeeper"), topicURI("monthly-close")),
	)

	add(decision(decisionURI("quickbooks-over-xero"), "Chose QuickBooks Online over Xero",
		"Selected QuickBooks Online over Xero for accounting software. Key factors: better bank feed integration with our credit union, more robust payroll add-on, and Rachel Green CPA already uses QuickBooks for other clients."))
	triples = append(triples,
		tripleHasTopic(decisionURI("quickbooks-over-xero"), topicURI("monthly-close")),
	)

	// Questions
	add(question(questionURI("bonus-depreciation"), "Can the espresso machine qualify for bonus depreciation in addition to Section 179, and what is the optimal strategy"))
	triples = append(triples,
		tripleHasTopic(questionURI("bonus-depreciation"), topicURI("depreciation")),
	)

	add(question(questionURI("quarterly-estimated-tax"), "What is the optimal schedule for quarterly estimated tax payments to minimize cash flow impact"))
	triples = append(triples,
		tripleHasTopic(questionURI("quarterly-estimated-tax"), topicURI("tax-planning")),
	)

	add(question(questionURI("tips-reporting"), "How to handle tips reporting for employees — cash tips vs credit card tips and proper W-2 reporting"))
	triples = append(triples,
		tripleHasTopic(questionURI("tips-reporting"), topicURI("payroll")),
		tripleTag(questionURI("tips-reporting"), tagURI("compliance")),
	)

	// Sources
	add(source(sourceURI("irs-pub-334"), "IRS Publication 334",
		"IRS Publication 334: Tax Guide for Small Business. Covers income, deductions, credits, and estimated tax payments for sole proprietors and small businesses."))
	triples = append(triples,
		tripleHasTopic(sourceURI("irs-pub-334"), topicURI("tax-planning")),
		tripleURL(sourceURI("irs-pub-334"), "https://www.irs.gov/publications/p334"),
	)

	add(source(sourceURI("oregon-biz-tax"), "Oregon DOR Business Tax Guide",
		"Oregon Department of Revenue business tax guide covering the Corporate Activity Tax (CAT), transient lodging tax, and quarterly filing requirements."))
	triples = append(triples,
		tripleHasTopic(sourceURI("oregon-biz-tax"), topicURI("tax-planning")),
		tripleURL(sourceURI("oregon-biz-tax"), "https://www.oregon.gov/dor/business"),
	)

	add(source(sourceURI("sba-loan-docs"), "SBA Loan Documentation Requirements",
		"SBA loan documentation requirements including financial statements, tax returns, business plan, and the requirement for accrual-basis accounting for loans above $250K."))
	triples = append(triples,
		tripleHasTopic(sourceURI("sba-loan-docs"), topicURI("revenue")),
	)

	// Cross-references
	triples = append(triples,
		tripleRef(noteURI("section-179"), entityURI("rachel-green")),
		tripleRef(decisionURI("accrual-accounting"), sourceURI("sba-loan-docs")),
		tripleRef(decisionURI("hire-bookkeeper"), entityURI("maria-santos")),
		tripleRef(decisionURI("quickbooks-over-xero"), entityURI("quickbooks-online")),
		tripleRef(questionURI("tips-reporting"), entityURI("maria-santos")),
		triplePartOf(entityURI("morning-brew-catering"), entityURI("riverside-cafe")),
		triplePartOf(entityURI("riverside-events"), entityURI("riverside-cafe")),
	)

	return testProject{
		Name:    "accounting",
		Graph:   "project:accounting",
		Triples: triples,
		Queries: []searchQuery{
			{
				Name:       "search revenue",
				Args:       map[string]any{"query": "revenue"},
				MinResults: 1,
				MustFind:   []string{topicURI("revenue"), noteURI("q1-summary")},
			},
			{
				Name:       "search Rachel Green CPA",
				Args:       map[string]any{"query": "Rachel Green CPA"},
				MinResults: 1,
				MustFind:   []string{entityURI("rachel-green")},
			},
			{
				Name:       "search Section 179 deduction",
				Args:       map[string]any{"query": "Section 179 deduction"},
				MinResults: 1,
				MustFind:   []string{noteURI("section-179")},
			},
			{
				Name:       "search monthly close checklist",
				Args:       map[string]any{"query": "monthly close checklist"},
				MinResults: 1,
				MustFind:   []string{noteURI("month-end-checklist")},
			},
			{
				Name:       "search QuickBooks",
				Args:       map[string]any{"query": "QuickBooks"},
				MinResults: 1,
				MustFind:   []string{entityURI("quickbooks-online"), decisionURI("quickbooks-over-xero")},
			},
			{
				Name:       "search espresso machine",
				Args:       map[string]any{"query": "espresso machine"},
				MinResults: 1,
				MustFind:   []string{noteURI("section-179"), questionURI("bonus-depreciation")},
			},
			{
				Name:       "search payroll",
				Args:       map[string]any{"query": "payroll"},
				MinResults: 1,
				MustFind:   []string{topicURI("payroll")},
			},
			{
				Name:       "search overdue AR accounts receivable",
				Args:       map[string]any{"query": "overdue accounts receivable"},
				MinResults: 1,
				MustFind:   []string{noteURI("q1-summary")},
			},
			{
				Name:       "search accrual accounting",
				Args:       map[string]any{"query": "accrual accounting"},
				MinResults: 1,
				MustFind:   []string{decisionURI("accrual-accounting")},
			},
			{
				Name:       "search Maria Santos bookkeeper",
				Args:       map[string]any{"query": "Maria Santos bookkeeper"},
				MinResults: 1,
				MustFind:   []string{entityURI("maria-santos")},
			},
			{
				Name:       "type filter decisions",
				Args:       map[string]any{"query": "accounting", "type": testHippoDecision},
				MinResults: 1,
				MustNotFind: []string{topicURI("revenue"), entityURI("quickbooks-online")},
			},
			{
				Name:       "search depreciation",
				Args:       map[string]any{"query": "depreciation"},
				MinResults: 1,
				MustFind:   []string{topicURI("depreciation")},
			},
			{
				Name:       "search Riverside Events",
				Args:       map[string]any{"query": "Riverside Events"},
				MinResults: 1,
				MustFind:   []string{entityURI("riverside-events")},
			},
			{
				Name:       "search tax planning",
				Args:       map[string]any{"query": "tax planning"},
				MinResults: 1,
				MustFind:   []string{topicURI("tax-planning")},
			},
			{
				Name:       "search bank reconciliation",
				Args:       map[string]any{"query": "bank reconciliation"},
				MinResults: 1,
				MustFind:   []string{noteURI("month-end-checklist")},
			},
			{
				Name:       "limit test",
				Args:       map[string]any{"query": "accounting", "limit": 2},
				MaxResults: 2,
			},
			{
				Name:       "negative search tomato",
				Args:       map[string]any{"query": "tomato"},
				MaxResults: 0,
			},
			{
				Name:       "case insensitive quickbooks",
				Args:       map[string]any{"query": "quickbooks"},
				MinResults: 1,
				MustFind:   []string{entityURI("quickbooks-online")},
			},
			{
				Name:       "search SBA loan",
				Args:       map[string]any{"query": "SBA loan"},
				MinResults: 1,
				MustFind:   []string{sourceURI("sba-loan-docs"), decisionURI("accrual-accounting")},
			},
			{
				Name:       "search chart of accounts",
				Args:       map[string]any{"query": "chart of accounts"},
				MinResults: 1,
				MustFind:   []string{noteURI("chart-of-accounts")},
			},
			{
				Name:       "search Morning Brew",
				Args:       map[string]any{"query": "Morning Brew"},
				MinResults: 1,
				MustFind:   []string{entityURI("morning-brew-catering")},
			},
			{
				Name:       "search tips reporting",
				Args:       map[string]any{"query": "tips reporting"},
				MinResults: 1,
				MustFind:   []string{questionURI("tips-reporting")},
			},
			{
				Name:       "search Oregon DOR",
				Args:       map[string]any{"query": "Oregon DOR"},
				MinResults: 1,
				MustFind:   []string{entityURI("oregon-dor"), sourceURI("oregon-biz-tax")},
			},
		},
	}
}
