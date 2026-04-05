package tools

func gardenProject() testProject {
	const (
		base = "https://hippocamp.dev/project/tomato-garden/"
		g    = base + "graph"

		// Topics
		topicVarieties   = base + "topic/varieties"
		topicSoil        = base + "topic/soil"
		topicPlanting    = base + "topic/planting"
		topicPestControl = base + "topic/pest-control"
		topicHarvest     = base + "topic/harvest"
		topicWatering    = base + "topic/watering"
		topicComposting  = base + "topic/composting"

		// Entities
		entityProject         = base + "entity/project"
		entitySanMarzano      = base + "entity/san-marzano"
		entityCherokeePurple  = base + "entity/cherokee-purple"
		entitySungold         = base + "entity/sungold"
		entityEarlyGirl       = base + "entity/early-girl"
		entityUncleJim        = base + "entity/uncle-jims-worm-farm"
		entityDripWorks       = base + "entity/dripworks"
		entityPortlandNursery = base + "entity/portland-nursery"

		// Notes
		noteSoilPrep    = base + "note/soil-prep"
		noteSeedStart   = base + "note/seed-starting"
		noteCompanion   = base + "note/companion-planting"
		noteDripSetup   = base + "note/drip-irrigation-setup"

		// Decisions
		decisionRaisedBeds = base + "decision/raised-beds"
		decisionOrganic    = base + "decision/organic-fertilizer"
		decisionDrip       = base + "decision/drip-irrigation"

		// Questions
		questionHornworm   = base + "question/hornworm-control"
		questionSuccession = base + "question/succession-planting"
		questionHardening  = base + "question/hardening-off"

		// Sources
		sourceOSU         = base + "source/osu-extension"
		sourceSunset      = base + "source/sunset-western-garden-book"
		sourceTerritorial = base + "source/territorial-seed-catalog"

		// Tags
		tagOrganic    = base + "tag/organic"
		tagSpringTask = base + "tag/spring-task"
		tagFallTask   = base + "tag/fall-task"
	)

	var triples []testTriple

	// Project entity
	triples = append(triples, entity(entityProject, "Tomato Garden Project", "Backyard tomato garden in Portland, OR focusing on heirloom and hybrid varieties")...)

	// Topics
	triples = append(triples, topic(topicVarieties, "Tomato Varieties", "Selection and comparison of tomato varieties for Pacific Northwest climate")...)
	triples = append(triples, topic(topicSoil, "Soil Preparation", "Soil amendments, testing, and bed preparation for optimal tomato growth")...)
	triples = append(triples, topic(topicPlanting, "Planting Schedule", "Seed starting, transplanting dates, and succession planting for Portland zone 8b")...)
	triples = append(triples, topic(topicPestControl, "Pest Control", "Managing common tomato pests including aphids, hornworms, and slugs organically")...)
	triples = append(triples, topic(topicHarvest, "Harvest and Storage", "Picking timing, ripening techniques, and preservation methods")...)
	triples = append(triples, topic(topicWatering, "Watering Schedule", "Irrigation timing, volume requirements, and watering best practices for tomatoes")...)
	triples = append(triples, topic(topicComposting, "Composting", "Compost production, worm composting, and organic matter recycling for garden soil")...)

	// Entities
	triples = append(triples, entity(entitySanMarzano, "San Marzano", "Classic Italian paste tomato, determinate, ideal for sauces and canning with meaty flesh")...)
	triples = append(triples, entity(entityCherokeePurple, "Cherokee Purple", "Heirloom beefsteak tomato with dusky purple skin, rich complex flavor, indeterminate")...)
	triples = append(triples, entity(entitySungold, "Sungold", "Hybrid cherry tomato producing intensely sweet orange fruit, prolific indeterminate vine")...)
	triples = append(triples, entity(entityEarlyGirl, "Early Girl", "Reliable slicer tomato that matures in 50 days, good for shorter Pacific Northwest seasons")...)
	triples = append(triples, entity(entityUncleJim, "Uncle Jim's Worm Farm", "Supplier of red wiggler composting worms and vermicomposting supplies")...)
	triples = append(triples, entity(entityDripWorks, "DripWorks", "Drip irrigation supplier providing tubing, emitters, timers, and complete garden kits")...)
	triples = append(triples, entity(entityPortlandNursery, "Portland Nursery", "Local nursery on SE Division carrying organic starts, soil amendments, and garden supplies")...)

	// Entity-topic relationships
	triples = append(triples, tripleHasTopic(entitySanMarzano, topicVarieties))
	triples = append(triples, tripleHasTopic(entityCherokeePurple, topicVarieties))
	triples = append(triples, tripleHasTopic(entitySungold, topicVarieties))
	triples = append(triples, tripleHasTopic(entityEarlyGirl, topicVarieties))
	triples = append(triples, tripleHasTopic(entityUncleJim, topicComposting))
	triples = append(triples, tripleHasTopic(entityDripWorks, topicWatering))
	triples = append(triples, tripleHasTopic(entityPortlandNursery, topicPlanting))

	// Notes
	triples = append(triples, note(noteSoilPrep, "Soil Preparation Notes", "Mix 1/3 compost, 1/3 native soil, 1/3 aged bark mulch. Target soil pH 6.2-6.8, add lime amendments if below 6.0. Work in bone meal for phosphorus and greensand for potassium.")...)
	triples = append(triples, note(noteSeedStart, "Indoor Seed Starting", "Start seeds indoors February 15 under grow lights, 6-8 weeks before last frost. Use heat mat at 75-80F for germination. Pot up to 4-inch containers when first true leaves appear.")...)
	triples = append(triples, note(noteCompanion, "Companion Planting Guide", "Plant basil between tomato plants to repel aphids and improve flavor. Marigolds around bed perimeter deter nematodes and whiteflies. Avoid planting near brassicas or fennel.")...)
	triples = append(triples, note(noteDripSetup, "Drip Irrigation Setup", "Install 1/2-inch mainline with 1/4-inch emitter tubing to each plant. Target 1 inch of water per week, delivered in 2-3 deep sessions. Timer set for early morning watering.")...)

	// Note-topic relationships
	triples = append(triples, tripleHasTopic(noteSoilPrep, topicSoil))
	triples = append(triples, tripleHasTopic(noteSeedStart, topicPlanting))
	triples = append(triples, tripleHasTopic(noteCompanion, topicPestControl))
	triples = append(triples, tripleHasTopic(noteDripSetup, topicWatering))

	// Note-entity references
	triples = append(triples, tripleRef(noteDripSetup, entityDripWorks))
	triples = append(triples, tripleRef(noteCompanion, entityPortlandNursery))

	// Decisions
	triples = append(triples, decision(decisionRaisedBeds, "Raised Beds Over In-Ground Planting", "Raised beds provide better drainage in Portland's wet climate, easier soil amendment control, warmer soil in spring, and reduced slug pressure compared to in-ground planting")...)
	triples = append(triples, decision(decisionOrganic, "Organic Over Synthetic Fertilizer", "Organic fertilizer chosen to build long-term soil health, support beneficial microorganisms, avoid chemical runoff, and maintain food safety for home consumption")...)
	triples = append(triples, decision(decisionDrip, "Drip Irrigation Over Sprinkler", "Drip irrigation selected for water efficiency, reduced foliar disease risk, consistent moisture delivery, and compatibility with raised bed layout")...)

	// Decision-topic relationships
	triples = append(triples, tripleHasTopic(decisionRaisedBeds, topicSoil))
	triples = append(triples, tripleHasTopic(decisionOrganic, topicSoil))
	triples = append(triples, tripleHasTopic(decisionDrip, topicWatering))

	// Questions
	triples = append(triples, question(questionHornworm, "What is the best organic method to control tomato hornworms without harming beneficial insects?")...)
	triples = append(triples, question(questionSuccession, "What is the optimal succession planting interval for continuous tomato harvest through October?")...)
	triples = append(triples, question(questionHardening, "When should seedlings start hardening off and what is the best gradual exposure schedule?")...)

	// Question-topic relationships
	triples = append(triples, tripleHasTopic(questionHornworm, topicPestControl))
	triples = append(triples, tripleHasTopic(questionSuccession, topicPlanting))
	triples = append(triples, tripleHasTopic(questionHardening, topicPlanting))

	// Sources
	triples = append(triples, source(sourceOSU, "OSU Extension Service", "Oregon State University Extension vegetable gardening guides covering Pacific Northwest growing conditions, pest management, and variety trials")...)
	triples = append(triples, source(sourceSunset, "Sunset Western Garden Book", "Comprehensive western US gardening reference with climate zone maps, plant encyclopedia, and regional growing calendars")...)
	triples = append(triples, source(sourceTerritorial, "Territorial Seed Company Catalog", "Seed catalog from Territorial Seed Company in Cottage Grove, OR with variety descriptions, days to maturity, and Pacific Northwest growing tips")...)

	// Source-topic relationships
	triples = append(triples, tripleHasTopic(sourceOSU, topicVarieties))
	triples = append(triples, tripleHasTopic(sourceSunset, topicPlanting))
	triples = append(triples, tripleHasTopic(sourceTerritorial, topicVarieties))

	// Source URLs
	triples = append(triples, tripleURL(sourceOSU, "https://extension.oregonstate.edu/vegetables"))
	triples = append(triples, tripleURL(sourceSunset, "https://www.sunsetmagazine.com/garden"))
	triples = append(triples, tripleURL(sourceTerritorial, "https://territorialseed.com"))

	// Tags
	triples = append(triples, tag(tagOrganic, "organic")...)
	triples = append(triples, tag(tagSpringTask, "spring-task")...)
	triples = append(triples, tag(tagFallTask, "fall-task")...)

	// Tag assignments
	triples = append(triples, tripleTag(decisionOrganic, tagOrganic))
	triples = append(triples, tripleTag(noteSoilPrep, tagOrganic))
	triples = append(triples, tripleTag(noteSeedStart, tagSpringTask))
	triples = append(triples, tripleTag(noteCompanion, tagSpringTask))
	triples = append(triples, tripleTag(noteSoilPrep, tagFallTask))

	// Part-of relationships
	triples = append(triples, triplePartOf(entitySanMarzano, entityProject))
	triples = append(triples, triplePartOf(entityCherokeePurple, entityProject))
	triples = append(triples, triplePartOf(entitySungold, entityProject))
	triples = append(triples, triplePartOf(entityEarlyGirl, entityProject))
	triples = append(triples, triplePartOf(entityDripWorks, entityProject))
	triples = append(triples, triplePartOf(entityPortlandNursery, entityProject))

	return testProject{
		Name:    "TomatoGarden",
		Graph:   g,
		Triples: triples,
		Queries: []searchQuery{
			{
				Name:       "tomato_varieties",
				Args:       map[string]any{"query": "tomato varieties"},
				MinResults: 1,
				MustFind:   []string{topicVarieties},
			},
			{
				Name:       "cherokee_purple",
				Args:       map[string]any{"query": "Cherokee Purple"},
				MinResults: 1,
				MustFind:   []string{entityCherokeePurple},
			},
			{
				Name:       "soil_ph",
				Args:       map[string]any{"query": "soil pH"},
				MinResults: 1,
				MustFind:   []string{noteSoilPrep},
			},
			{
				Name:       "pest_aphid",
				Args:       map[string]any{"query": "pest aphid"},
				MinResults: 1,
				MustFind:   []string{topicPestControl},
			},
			{
				Name:       "drip_irrigation",
				Args:       map[string]any{"query": "drip irrigation"},
				MinResults: 1,
				MustFind:   []string{entityDripWorks},
			},
			{
				Name:       "companion_planting",
				Args:       map[string]any{"query": "companion planting"},
				MinResults: 1,
				MustFind:   []string{noteCompanion},
			},
			{
				Name:       "raised_beds",
				Args:       map[string]any{"query": "raised beds"},
				MinResults: 1,
				MustFind:   []string{decisionRaisedBeds},
			},
			{
				Name:       "organic_fertilizer",
				Args:       map[string]any{"query": "organic fertilizer"},
				MinResults: 1,
				MustFind:   []string{decisionOrganic},
			},
			{
				Name:       "hornworm",
				Args:       map[string]any{"query": "hornworm"},
				MinResults: 1,
				MustFind:   []string{questionHornworm},
			},
			{
				Name:       "seed_starting_february",
				Args:       map[string]any{"query": "seed starting February"},
				MinResults: 1,
				MustFind:   []string{noteSeedStart},
			},
			{
				Name:       "territorial_seed",
				Args:       map[string]any{"query": "Territorial Seed"},
				MinResults: 1,
				MustFind:   []string{sourceTerritorial},
			},
			{
				Name:       "harvest",
				Args:       map[string]any{"query": "harvest"},
				MinResults: 1,
				MustFind:   []string{topicHarvest},
			},
			{
				Name:       "tomato_type_filter",
				Args:       map[string]any{"query": "tomato", "type": testHippoEntity},
				MinResults: 1,
				MustFind:   []string{entitySanMarzano},
			},
			{
				Name:       "sungold_cherry",
				Args:       map[string]any{"query": "Sungold cherry"},
				MinResults: 1,
				MustFind:   []string{entitySungold},
			},
			{
				Name:       "watering_schedule",
				Args:       map[string]any{"query": "watering schedule"},
				MinResults: 1,
				MustFind:   []string{topicWatering},
			},
			{
				Name:       "composting_worm",
				Args:       map[string]any{"query": "composting worm"},
				MinResults: 1,
				MustFind:   []string{entityUncleJim, topicComposting},
			},
			{
				Name:       "marigold_basil",
				Args:       map[string]any{"query": "marigold basil"},
				MinResults: 1,
				MustFind:   []string{noteCompanion},
			},
			{
				Name:       "garden_limit",
				Args:       map[string]any{"query": "garden", "limit": 2},
				MinResults: 1,
				MaxResults: 2,
			},
			{
				Name:       "negative_electrical_permit",
				Args:       map[string]any{"query": "electrical permit"},
				MaxResults: 0,
			},
			{
				Name:       "case_insensitive_cherokee",
				Args:       map[string]any{"query": "CHEROKEE"},
				MinResults: 1,
				MustFind:   []string{entityCherokeePurple},
			},
			{
				Name:       "portland_nursery",
				Args:       map[string]any{"query": "Portland Nursery"},
				MinResults: 1,
				MustFind:   []string{entityPortlandNursery},
			},
			{
				Name:       "succession_planting",
				Args:       map[string]any{"query": "succession planting"},
				MinResults: 1,
				MustFind:   []string{questionSuccession},
			},
			{
				Name:       "osu_extension",
				Args:       map[string]any{"query": "OSU Extension"},
				MinResults: 1,
				MustFind:   []string{sourceOSU},
			},
		},
	}
}
