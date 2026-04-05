package tools

func salesProject() testProject {
	const base = "https://hippocamp.dev/project/sales-department/"

	// URI helpers for this project.
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
	add(topic(topicURI("pipeline"), "Pipeline", "Sales pipeline tracking and management"))
	add(topic(topicURI("clients"), "Clients", "Current and prospective client accounts"))
	add(topic(topicURI("targets"), "Targets", "Quarterly and annual sales targets"))
	add(topic(topicURI("methodology"), "Sales Methodology", "Frameworks and processes for selling"))
	add(topic(topicURI("competitive-analysis"), "Competitive Analysis", "Analysis of competing products and vendors"))
	add(topic(topicURI("pricing"), "Pricing", "Product pricing strategy and structures"))

	// Tags
	add(tag(tagURI("enterprise"), "enterprise"))
	add(tag(tagURI("prospect"), "prospect"))
	add(tag(tagURI("high-value"), "high-value"))

	// Entities
	add(entity(entityURI("acme-corp"), "Acme Corp", "Enterprise client, Lisa Wang VP Engineering, $180K ARR, 500-seat deployment"))
	triples = append(triples,
		tripleHasTopic(entityURI("acme-corp"), topicURI("clients")),
		tripleTag(entityURI("acme-corp"), tagURI("enterprise")),
	)

	add(entity(entityURI("globex-industries"), "Globex Industries", "Mid-market client, David Park CTO, $85K ARR, requires SAP integration"))
	triples = append(triples,
		tripleHasTopic(entityURI("globex-industries"), topicURI("clients")),
		tripleTag(entityURI("globex-industries"), tagURI("enterprise")),
	)

	add(entity(entityURI("initech"), "Initech", "Prospective client, Michael Chen Head of IT, $120K prospect, evaluating CloudSync Pro"))
	triples = append(triples,
		tripleHasTopic(entityURI("initech"), topicURI("clients")),
		tripleTag(entityURI("initech"), tagURI("prospect")),
	)

	add(entity(entityURI("jennifer-torres"), "Jennifer Torres", "VP of Sales, leads the sales team and sets quarterly targets"))
	triples = append(triples,
		tripleHasTopic(entityURI("jennifer-torres"), topicURI("targets")),
	)

	add(entity(entityURI("cloudsync-pro"), "CloudSync Pro", "Our flagship SaaS product for enterprise data synchronization"))
	triples = append(triples,
		tripleHasTopic(entityURI("cloudsync-pro"), topicURI("pipeline")),
	)

	add(entity(entityURI("dataflow"), "DataFlow", "Primary competitor, aggressive pricing model, targets mid-market"))
	triples = append(triples,
		tripleHasTopic(entityURI("dataflow"), topicURI("competitive-analysis")),
	)

	add(entity(entityURI("syncwave"), "SyncWave", "Competitor focused on small business segment with freemium model"))
	triples = append(triples,
		tripleHasTopic(entityURI("syncwave"), topicURI("competitive-analysis")),
	)

	add(entity(entityURI("meridian-group"), "Meridian Group", "Enterprise prospect, $200K deal in negotiation, Fortune 500 company"))
	triples = append(triples,
		tripleHasTopic(entityURI("meridian-group"), topicURI("clients")),
		tripleTag(entityURI("meridian-group"), tagURI("high-value")),
	)

	// Notes
	add(note(noteURI("meddic-framework"), "MEDDIC Qualification Framework",
		"MEDDIC qualification framework for enterprise deals: Metrics, Economic Buyer, Decision Criteria, Decision Process, Identify Pain, Champion. Each opportunity must score above 60% before advancing to proposal stage."))
	triples = append(triples,
		tripleHasTopic(noteURI("meddic-framework"), topicURI("methodology")),
	)

	add(note(noteURI("demo-script"), "Demo Script Outline",
		"Standard demo script for CloudSync Pro: intro (2 min), pain discovery (5 min), live demo of sync engine (10 min), ROI calculator walkthrough (5 min), Q&A and next steps (8 min). Total 30 minutes."))
	triples = append(triples,
		tripleHasTopic(noteURI("demo-script"), topicURI("methodology")),
	)

	add(note(noteURI("objection-handling"), "Objection Handling Guide",
		"Common objections and responses: Price — show TCO comparison over 3 years. Security — present SOC2 Type II and penetration test results. Migration — offer white-glove migration service with dedicated engineer."))
	triples = append(triples,
		tripleHasTopic(noteURI("objection-handling"), topicURI("methodology")),
	)

	add(note(noteURI("pipeline-metrics"), "Pipeline Metrics Q2",
		"Current pipeline: $3.1M total weighted value, 47 opportunities, 12 in negotiation stage. Average deal size $66K. Win rate 34%. Average sales cycle 68 days."))
	triples = append(triples,
		tripleHasTopic(noteURI("pipeline-metrics"), topicURI("pipeline")),
	)

	add(note(noteURI("commission-structure"), "Commission Structure",
		"Base commission 8% of ACV. Accelerators above 110% quota: 12% for 110-130%, 16% for 130%+. SPIFFs for multi-year deals: extra 2% for 2-year, 4% for 3-year commitments."))
	triples = append(triples,
		tripleHasTopic(noteURI("commission-structure"), topicURI("targets")),
	)

	// Decisions
	add(decision(decisionURI("adopt-meddic"), "Adopted MEDDIC over SPIN Selling",
		"MEDDIC provides better qualification criteria for enterprise deals over $50K. SPIN works well for SMB but lacks the rigor needed for complex multi-stakeholder sales."))
	triples = append(triples,
		tripleHasTopic(decisionURI("adopt-meddic"), topicURI("methodology")),
	)

	add(decision(decisionURI("annual-billing"), "Chose Annual Billing over Monthly",
		"Annual billing reduces churn by 40% compared to monthly. Customers on annual plans have 2.3x higher LTV. Offering 2-month discount for annual commitment."))
	triples = append(triples,
		tripleHasTopic(decisionURI("annual-billing"), topicURI("pricing")),
	)

	add(decision(decisionURI("salesforce-crm"), "Selected Salesforce over HubSpot",
		"Salesforce chosen for CRM due to enterprise feature set, CPQ module, and integration with existing ERP. HubSpot lacked advanced forecasting and territory management."))
	triples = append(triples,
		tripleHasTopic(decisionURI("salesforce-crm"), topicURI("pipeline")),
	)

	// Questions
	add(question(questionURI("dataflow-pricing"), "How to position against DataFlow's aggressive pricing — they undercut us by 30% on mid-market deals"))
	triples = append(triples,
		tripleHasTopic(questionURI("dataflow-pricing"), topicURI("competitive-analysis")),
	)

	add(question(questionURI("globex-sap"), "Is Globex SAP integration feasible within their Q3 timeline and our current engineering bandwidth"))
	triples = append(triples,
		tripleHasTopic(questionURI("globex-sap"), topicURI("clients")),
	)

	add(question(questionURI("territory-expansion"), "Q3 territory expansion — should we enter the Southeast region or double down on the West Coast"))
	triples = append(triples,
		tripleHasTopic(questionURI("territory-expansion"), topicURI("targets")),
	)

	// Sources
	add(source(sourceURI("gartner-mq"), "Gartner Magic Quadrant for Data Integration",
		"Gartner Magic Quadrant 2025 for Data Integration Tools. CloudSync Pro positioned as a Visionary. DataFlow is a Niche Player."))
	triples = append(triples,
		tripleHasTopic(sourceURI("gartner-mq"), topicURI("competitive-analysis")),
		tripleURL(sourceURI("gartner-mq"), "https://www.gartner.com/mq/data-integration-2025"),
	)

	add(source(sourceURI("win-loss-analysis"), "Internal Win/Loss Analysis Q1",
		"Win/loss analysis of 83 closed opportunities. Top win reasons: product reliability, customer support. Top loss reasons: pricing, lack of specific integrations."))
	triples = append(triples,
		tripleHasTopic(sourceURI("win-loss-analysis"), topicURI("pipeline")),
	)

	add(source(sourceURI("competitive-intel"), "Competitive Intelligence Report",
		"Competitive intelligence report covering DataFlow and SyncWave feature roadmaps, pricing changes, and recent customer wins."))
	triples = append(triples,
		tripleHasTopic(sourceURI("competitive-intel"), topicURI("competitive-analysis")),
	)

	// Cross-references
	triples = append(triples,
		tripleRef(entityURI("acme-corp"), entityURI("cloudsync-pro")),
		tripleRef(entityURI("initech"), entityURI("cloudsync-pro")),
		tripleRef(entityURI("meridian-group"), entityURI("cloudsync-pro")),
		tripleRef(noteURI("meddic-framework"), decisionURI("adopt-meddic")),
		tripleRef(questionURI("dataflow-pricing"), entityURI("dataflow")),
		tripleRef(questionURI("globex-sap"), entityURI("globex-industries")),
	)

	return testProject{
		Name:    "sales-department",
		Graph:   "project:sales-department",
		Triples: triples,
		Queries: []searchQuery{
			{
				Name:       "search Acme Corp",
				Args:       map[string]any{"query": "Acme Corp"},
				MinResults: 1,
				MustFind:   []string{entityURI("acme-corp")},
			},
			{
				Name:       "search Lisa Wang",
				Args:       map[string]any{"query": "Lisa Wang"},
				MinResults: 1,
				MustFind:   []string{entityURI("acme-corp")},
			},
			{
				Name:       "search pipeline metrics",
				Args:       map[string]any{"query": "pipeline metrics"},
				MinResults: 1,
				MustFind:   []string{noteURI("pipeline-metrics")},
			},
			{
				Name:       "search MEDDIC",
				Args:       map[string]any{"query": "MEDDIC"},
				MinResults: 1,
				MustFind:   []string{noteURI("meddic-framework"), decisionURI("adopt-meddic")},
			},
			{
				Name:       "search DataFlow competitor",
				Args:       map[string]any{"query": "DataFlow competitor"},
				MinResults: 1,
				MustFind:   []string{entityURI("dataflow")},
			},
			{
				Name:       "search commission",
				Args:       map[string]any{"query": "commission"},
				MinResults: 1,
				MustFind:   []string{noteURI("commission-structure")},
			},
			{
				Name:       "search SAP integration",
				Args:       map[string]any{"query": "SAP integration"},
				MinResults: 1,
				MustFind:   []string{entityURI("globex-industries"), questionURI("globex-sap")},
			},
			{
				Name:       "search Jennifer Torres",
				Args:       map[string]any{"query": "Jennifer Torres"},
				MinResults: 1,
				MustFind:   []string{entityURI("jennifer-torres")},
			},
			{
				Name:       "type filter entities",
				Args:       map[string]any{"query": "client", "type": testHippoEntity},
				MinResults: 1,
				MustNotFind: []string{noteURI("pipeline-metrics"), topicURI("clients")},
			},
			{
				Name:       "search annual billing decision",
				Args:       map[string]any{"query": "annual billing"},
				MinResults: 1,
				MustFind:   []string{decisionURI("annual-billing")},
			},
			{
				Name:       "search Salesforce decision",
				Args:       map[string]any{"query": "Salesforce"},
				MinResults: 1,
				MustFind:   []string{decisionURI("salesforce-crm")},
			},
			{
				Name:       "search Gartner",
				Args:       map[string]any{"query": "Gartner"},
				MinResults: 1,
				MustFind:   []string{sourceURI("gartner-mq")},
			},
			{
				Name:       "limit test",
				Args:       map[string]any{"query": "sales", "limit": 3},
				MaxResults: 3,
			},
			{
				Name:       "negative search tomato",
				Args:       map[string]any{"query": "tomato"},
				MaxResults: 0,
			},
			{
				Name:       "case insensitive meddic",
				Args:       map[string]any{"query": "meddic"},
				MinResults: 1,
				MustFind:   []string{noteURI("meddic-framework")},
			},
			{
				Name:       "search CloudSync Pro",
				Args:       map[string]any{"query": "CloudSync Pro"},
				MinResults: 1,
				MustFind:   []string{entityURI("cloudsync-pro")},
			},
			{
				Name:       "search Meridian Group",
				Args:       map[string]any{"query": "Meridian Group"},
				MinResults: 1,
				MustFind:   []string{entityURI("meridian-group")},
			},
			{
				Name:       "search objection handling",
				Args:       map[string]any{"query": "objection handling"},
				MinResults: 1,
				MustFind:   []string{noteURI("objection-handling")},
			},
			{
				Name:       "search demo script",
				Args:       map[string]any{"query": "demo script"},
				MinResults: 1,
				MustFind:   []string{noteURI("demo-script")},
			},
			{
				Name:       "search territory expansion",
				Args:       map[string]any{"query": "territory expansion"},
				MinResults: 1,
				MustFind:   []string{questionURI("territory-expansion")},
			},
			{
				Name:       "search pricing",
				Args:       map[string]any{"query": "pricing"},
				MinResults: 1,
				MustFind:   []string{topicURI("pricing")},
			},
			{
				Name:       "search Michael Chen Initech",
				Args:       map[string]any{"query": "Michael Chen Initech"},
				MinResults: 1,
				MustFind:   []string{entityURI("initech")},
			},
		},
	}
}
