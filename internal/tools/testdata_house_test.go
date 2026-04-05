package tools

func houseProject() testProject {
	const (
		base = "https://hippocamp.dev/project/house-construction/"
		g    = base + "graph"

		// Topics
		topicPermits    = base + "topic/permits"
		topicMaterials  = base + "topic/materials"
		topicBudget     = base + "topic/budget"
		topicSchedule   = base + "topic/schedule"
		topicElectrical = base + "topic/electrical"
		topicPlumbing   = base + "topic/plumbing"
		topicRoofing    = base + "topic/roofing"
		topicFoundation = base + "topic/foundation"
		topicHVAC       = base + "topic/hvac"

		// Entities
		entityLoneStar    = base + "entity/lone-star-builders"
		entityBrightSpark = base + "entity/bright-spark-electric"
		entityFerguson    = base + "entity/ferguson-supply"
		entityHillCountry = base + "entity/hill-country-lumber"
		entityComfortAir  = base + "entity/comfort-air-solutions"
		entityMaria       = base + "entity/maria-rodriguez"
		entityProject     = base + "entity/project"

		// Notes
		noteFoundation = base + "note/foundation-specs"
		noteLumber     = base + "note/lumber-order"
		notePlumbing   = base + "note/low-flow-fixtures"

		// Decisions
		decisionRoof       = base + "decision/metal-roof"
		decisionInsulation = base + "decision/spray-foam-insulation"
		decisionWaterHeater = base + "decision/tankless-water-heater"

		// Questions
		questionDrainage = base + "question/drainage-east-side"
		questionHVAC     = base + "question/hvac-ductwork-routing"
		questionPermit   = base + "question/permit-timeline"

		// Sources
		sourceAustinCodes = base + "source/austin-building-codes"
		sourceNEC         = base + "source/nec-2023"
		sourceIRC         = base + "source/international-residential-code"

		// Tags
		tagUrgent          = base + "tag/urgent"
		tagBudgetCritical  = base + "tag/budget-critical"
		tagWeatherDependent = base + "tag/weather-dependent"
	)

	var triples []testTriple

	// Project entity
	triples = append(triples, entity(entityProject, "House Construction Project", "New single-family home construction in Austin, TX")...)

	// Topics
	triples = append(triples, topic(topicPermits, "Building Permits", "Permits required for new residential construction in Austin, TX")...)
	triples = append(triples, topic(topicMaterials, "Construction Materials", "Materials sourcing and procurement for the build")...)
	triples = append(triples, topic(topicBudget, "Budget and Cost Tracking", "Overall project budget, cost estimates, and expense tracking")...)
	triples = append(triples, topic(topicSchedule, "Construction Schedule", "Project timeline, milestones, and phase scheduling")...)
	triples = append(triples, topic(topicElectrical, "Electrical Systems", "Electrical wiring, panels, and fixture installation")...)
	triples = append(triples, topic(topicPlumbing, "Plumbing Systems", "Plumbing rough-in, fixtures, and water supply lines")...)
	triples = append(triples, topic(topicRoofing, "Roofing", "Roof structure, materials, and installation")...)
	triples = append(triples, topic(topicFoundation, "Foundation", "Foundation type, specifications, and construction")...)
	triples = append(triples, topic(topicHVAC, "HVAC Systems", "Heating, ventilation, and air conditioning design and installation")...)

	// Entities
	triples = append(triples, entity(entityLoneStar, "Lone Star Builders", "General contractor managed by Jim Patterson, handling overall construction coordination")...)
	triples = append(triples, entity(entityBrightSpark, "Bright Spark Electric", "Electrical subcontractor led by Tom Chen, licensed for residential and commercial work")...)
	triples = append(triples, entity(entityFerguson, "Ferguson Supply", "Plumbing fixture and supply distributor providing low-flow fixtures and pipe fittings")...)
	triples = append(triples, entity(entityHillCountry, "Hill Country Lumber", "Local lumber yard supplying framing materials and Douglas Fir structural lumber")...)
	triples = append(triples, entity(entityComfortAir, "Comfort Air Solutions", "HVAC contractor managed by Sarah Kim, specializing in high-efficiency residential systems")...)
	triples = append(triples, entity(entityMaria, "Maria Rodriguez", "City building inspector responsible for foundation, framing, and final inspections")...)

	// Entity-topic relationships
	triples = append(triples, tripleHasTopic(entityLoneStar, topicSchedule))
	triples = append(triples, tripleHasTopic(entityBrightSpark, topicElectrical))
	triples = append(triples, tripleHasTopic(entityFerguson, topicPlumbing))
	triples = append(triples, tripleHasTopic(entityHillCountry, topicMaterials))
	triples = append(triples, tripleHasTopic(entityComfortAir, topicHVAC))
	triples = append(triples, tripleHasTopic(entityMaria, topicPermits))

	// Notes
	triples = append(triples, note(noteFoundation, "Foundation Specifications", "Concrete slab foundation with 4-inch minimum thickness, reinforced with #4 rebar grid at 12-inch spacing. Soil test confirmed expansive clay requiring post-tension cables.")...)
	triples = append(triples, note(noteLumber, "Lumber Order Details", "Douglas Fir 2x6 framing lumber, 4,200 board feet total. Delivery schedule: first load April 15, second load May 1. Hill Country Lumber confirmed pricing.")...)
	triples = append(triples, note(notePlumbing, "Low-Flow Fixture Requirements", "All fixtures must meet WaterSense certification. Low-flow toilets at 1.28 GPF, faucets at 1.5 GPM. Ferguson Supply providing complete fixture package.")...)

	// Note-topic relationships
	triples = append(triples, tripleHasTopic(noteFoundation, topicFoundation))
	triples = append(triples, tripleHasTopic(noteLumber, topicMaterials))
	triples = append(triples, tripleHasTopic(notePlumbing, topicPlumbing))

	// Note-entity references
	triples = append(triples, tripleRef(noteLumber, entityHillCountry))
	triples = append(triples, tripleRef(notePlumbing, entityFerguson))

	// Decisions
	triples = append(triples, decision(decisionRoof, "Metal Roof Over Asphalt Shingles", "Standing seam metal roof selected for longevity (50+ year lifespan), superior hail resistance critical in central Texas, and better energy efficiency with reflective coating")...)
	triples = append(triples, decision(decisionInsulation, "Spray Foam Insulation", "Closed-cell spray foam chosen for superior energy efficiency, air sealing properties, and moisture barrier capability in humid Austin climate")...)
	triples = append(triples, decision(decisionWaterHeater, "Tankless Water Heater", "Tankless unit selected for space saving in utility closet, on-demand heating reduces energy waste, and longer lifespan compared to tank models")...)

	// Decision-topic relationships
	triples = append(triples, tripleHasTopic(decisionRoof, topicRoofing))
	triples = append(triples, tripleHasTopic(decisionInsulation, topicMaterials))
	triples = append(triples, tripleHasTopic(decisionWaterHeater, topicPlumbing))

	// Questions
	triples = append(triples, question(questionDrainage, "What is the best drainage solution for the east side of the lot where water pools after heavy rain?")...)
	triples = append(triples, question(questionHVAC, "How should HVAC ductwork be routed to minimize ceiling bulkheads in the open floor plan?")...)
	triples = append(triples, question(questionPermit, "What is the expected permit timeline for new residential construction in Austin?")...)

	// Question-topic relationships
	triples = append(triples, tripleHasTopic(questionDrainage, topicFoundation))
	triples = append(triples, tripleHasTopic(questionHVAC, topicHVAC))
	triples = append(triples, tripleHasTopic(questionPermit, topicPermits))

	// Sources
	triples = append(triples, source(sourceAustinCodes, "Austin Building Codes", "City of Austin residential building code requirements including setbacks, height limits, and energy compliance")...)
	triples = append(triples, source(sourceNEC, "NEC 2023", "National Electrical Code 2023 edition covering residential wiring standards, AFCI/GFCI requirements, and panel sizing")...)
	triples = append(triples, source(sourceIRC, "International Residential Code", "IRC standards for single-family residential construction including structural, plumbing, and mechanical requirements")...)

	// Source-topic relationships
	triples = append(triples, tripleHasTopic(sourceAustinCodes, topicPermits))
	triples = append(triples, tripleHasTopic(sourceNEC, topicElectrical))
	triples = append(triples, tripleHasTopic(sourceIRC, topicFoundation))

	// Source URLs
	triples = append(triples, tripleURL(sourceAustinCodes, "https://www.austintexas.gov/building-codes"))
	triples = append(triples, tripleURL(sourceNEC, "https://www.nfpa.org/nec"))
	triples = append(triples, tripleURL(sourceIRC, "https://www.iccsafe.org/irc"))

	// Tags
	triples = append(triples, tag(tagUrgent, "urgent")...)
	triples = append(triples, tag(tagBudgetCritical, "budget-critical")...)
	triples = append(triples, tag(tagWeatherDependent, "weather-dependent")...)

	// Tag assignments
	triples = append(triples, tripleTag(noteFoundation, tagUrgent))
	triples = append(triples, tripleTag(decisionRoof, tagBudgetCritical))
	triples = append(triples, tripleTag(noteLumber, tagWeatherDependent))
	triples = append(triples, tripleTag(decisionRoof, tagWeatherDependent))

	// Part-of relationships
	triples = append(triples, triplePartOf(entityLoneStar, entityProject))
	triples = append(triples, triplePartOf(entityBrightSpark, entityProject))
	triples = append(triples, triplePartOf(entityFerguson, entityProject))
	triples = append(triples, triplePartOf(entityHillCountry, entityProject))
	triples = append(triples, triplePartOf(entityComfortAir, entityProject))

	return testProject{
		Name:    "HouseConstruction",
		Graph:   g,
		Triples: triples,
		Queries: []searchQuery{
			{
				Name:       "building_permit",
				Args:       map[string]any{"query": "building permit"},
				MinResults: 1,
				MustFind:   []string{topicPermits},
			},
			{
				Name:       "budget_cost",
				Args:       map[string]any{"query": "budget cost"},
				MinResults: 1,
				MustFind:   []string{topicBudget},
			},
			{
				Name:       "lumber",
				Args:       map[string]any{"query": "lumber"},
				MinResults: 1,
				MustFind:   []string{entityHillCountry},
			},
			{
				Name:        "electrical_precision",
				Args:        map[string]any{"query": "electrical"},
				MinResults:  1,
				MustFind:    []string{entityBrightSpark},
				MustNotFind: []string{entityFerguson},
			},
			{
				Name:       "jim_patterson",
				Args:       map[string]any{"query": "Jim Patterson"},
				MinResults: 1,
				MustFind:   []string{entityLoneStar},
			},
			{
				Name:       "inspector",
				Args:       map[string]any{"query": "inspector"},
				MinResults: 1,
				MustFind:   []string{entityMaria},
			},
			{
				Name:       "metal_roof",
				Args:       map[string]any{"query": "metal roof"},
				MinResults: 1,
				MustFind:   []string{decisionRoof},
			},
			{
				Name:       "insulation",
				Args:       map[string]any{"query": "insulation"},
				MinResults: 1,
				MustFind:   []string{decisionInsulation},
			},
			{
				Name:       "foundation_concrete",
				Args:       map[string]any{"query": "foundation concrete"},
				MinResults: 1,
				MustFind:   []string{noteFoundation},
			},
			{
				Name:       "hvac_comfort_air",
				Args:       map[string]any{"query": "HVAC"},
				MinResults: 1,
				MustFind:   []string{entityComfortAir},
			},
			{
				Name:       "plumbing_fixture",
				Args:       map[string]any{"query": "plumbing fixture"},
				MinResults: 1,
				MustFind:   []string{entityFerguson, topicPlumbing},
			},
			{
				Name:       "nec_2023",
				Args:       map[string]any{"query": "NEC 2023"},
				MinResults: 1,
				MustFind:   []string{sourceNEC},
			},
			{
				Name:       "contractor_type_filter",
				Args:       map[string]any{"query": "contractor", "type": testHippoEntity},
				MinResults: 1,
				MustFind:   []string{entityLoneStar},
			},
			{
				Name:       "drainage_east",
				Args:       map[string]any{"query": "drainage east"},
				MinResults: 1,
				MustFind:   []string{questionDrainage},
			},
			{
				Name:       "budget_critical_tag",
				Args:       map[string]any{"query": "budget-critical"},
				MinResults: 1,
				MustFind:   []string{tagBudgetCritical},
			},
			{
				Name:       "case_insensitive_lumber",
				Args:       map[string]any{"query": "LUMBER"},
				MinResults: 1,
				MustFind:   []string{entityHillCountry},
			},
			{
				Name:       "delivery_schedule",
				Args:       map[string]any{"query": "delivery schedule"},
				MinResults: 1,
				MustFind:   []string{noteLumber},
			},
			{
				Name:       "construction_limit",
				Args:       map[string]any{"query": "construction", "limit": 3},
				MinResults: 1,
				MaxResults: 3,
			},
			{
				Name:       "tankless_water_heater",
				Args:       map[string]any{"query": "tankless water heater"},
				MinResults: 1,
				MustFind:   []string{decisionWaterHeater},
			},
			{
				Name:       "negative_tomato",
				Args:       map[string]any{"query": "tomato"},
				MaxResults: 0,
			},
			{
				Name:       "douglas_fir",
				Args:       map[string]any{"query": "Douglas Fir"},
				MinResults: 1,
				MustFind:   []string{noteLumber},
			},
			{
				Name:       "weather_dependent_tag",
				Args:       map[string]any{"query": "weather-dependent"},
				MinResults: 1,
				MustFind:   []string{tagWeatherDependent},
			},
			{
				Name:       "sarah_kim",
				Args:       map[string]any{"query": "Sarah Kim"},
				MinResults: 1,
				MustFind:   []string{entityComfortAir},
			},
		},
	}
}
