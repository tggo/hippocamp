package tools

func recipesProject() testProject {
	const base = "https://hippocamp.dev/project/recipe-collection/"

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
	add(topic(topicURI("baking"), "Baking", "Bread, pastry, and baked goods techniques and recipes"))
	add(topic(topicURI("main-courses"), "Main Courses", "Entrees and main dish recipes"))
	add(topic(topicURI("desserts"), "Desserts", "Sweet dishes, cakes, and confections"))
	add(topic(topicURI("fermentation"), "Fermentation", "Sourdough starters, fermented foods, and wild yeast"))
	add(topic(topicURI("knife-skills"), "Knife Skills", "Cutting techniques and knife maintenance"))
	add(topic(topicURI("meal-planning"), "Meal Planning", "Weekly meal prep and planning strategies"))
	add(topic(topicURI("ingredients"), "Ingredients", "Flour types, spices, and sourcing quality ingredients"))

	// Tags
	add(tag(tagURI("vegetarian"), "vegetarian"))
	add(tag(tagURI("advanced"), "advanced"))
	add(tag(tagURI("essential"), "essential"))

	// Entities
	add(entity(entityURI("king-arthur-flour"), "King Arthur Flour", "Premium flour brand, preferred for bread baking, consistent protein content across batches"))
	triples = append(triples,
		tripleHasTopic(entityURI("king-arthur-flour"), topicURI("ingredients")),
	)

	add(entity(entityURI("bobs-red-mill"), "Bob's Red Mill", "Specialty grain and flour supplier, excellent for whole wheat, rye, and alternative flours"))
	triples = append(triples,
		tripleHasTopic(entityURI("bobs-red-mill"), topicURI("ingredients")),
	)

	add(entity(entityURI("burlap-and-barrel"), "Burlap & Barrel", "Single-origin spice company, exceptional quality za'atar, paprika, and cumin"))
	triples = append(triples,
		tripleHasTopic(entityURI("burlap-and-barrel"), topicURI("ingredients")),
	)

	add(entity(entityURI("penzeys"), "Penzeys Spices", "Spice retailer with wide selection of blends and individual spices, good value for bulk"))
	triples = append(triples,
		tripleHasTopic(entityURI("penzeys"), topicURI("ingredients")),
	)

	add(entity(entityURI("madhur-jaffrey"), "Madhur Jaffrey", "Acclaimed cookbook author and authority on Indian cooking, known for accessible recipes"))
	triples = append(triples,
		tripleHasTopic(entityURI("madhur-jaffrey"), topicURI("main-courses")),
	)

	add(entity(entityURI("dutch-oven"), "Dutch Oven", "Essential equipment for bread baking and braising, cast iron with lid, retains steam for crust development"))
	triples = append(triples,
		tripleHasTopic(entityURI("dutch-oven"), topicURI("baking")),
		tripleTag(entityURI("dutch-oven"), tagURI("essential")),
	)

	// Notes
	add(note(noteURI("sourdough-recipe"), "Sourdough Bread Recipe",
		"Basic sourdough: 500g bread flour, 350g water (70% hydration), 100g active starter, 10g salt. Autolyse 30 min. Bulk ferment 4-6 hours with stretch and fold every 30 min for first 2 hours. Shape, cold retard 12-16 hours. Bake in Dutch oven at 500F lid on 20 min, lid off 25 min."))
	triples = append(triples,
		tripleHasTopic(noteURI("sourdough-recipe"), topicURI("baking")),
		tripleHasTopic(noteURI("sourdough-recipe"), topicURI("fermentation")),
		tripleTag(noteURI("sourdough-recipe"), tagURI("vegetarian")),
	)

	add(note(noteURI("chicken-tikka-masala"), "Chicken Tikka Masala",
		"Chicken tikka masala: marinate chicken in yogurt, garam masala, turmeric, cumin, and chili powder for 4 hours. Grill or broil until charred. Simmer in sauce of tomatoes, cream, fenugreek leaves, and spices. Serve with basmati rice and naan."))
	triples = append(triples,
		tripleHasTopic(noteURI("chicken-tikka-masala"), topicURI("main-courses")),
	)

	add(note(noteURI("tiramisu"), "Tiramisu Recipe",
		"Classic tiramisu with mascarpone cream, no raw eggs. Whip mascarpone with sugar and vanilla. Fold in whipped cream. Layer espresso-dipped ladyfingers with mascarpone mixture. Refrigerate 6 hours minimum. Dust with cocoa powder before serving."))
	triples = append(triples,
		tripleHasTopic(noteURI("tiramisu"), topicURI("desserts")),
		tripleTag(noteURI("tiramisu"), tagURI("vegetarian")),
	)

	add(note(noteURI("starter-maintenance"), "Sourdough Starter Maintenance",
		"Starter maintenance: feed at 1:1:1 ratio (starter:flour:water) every 12 hours at room temp, or weekly if refrigerated. Use a mix of whole wheat and AP flour for best activity. Starter should double in 4-6 hours when ready to use. Discard can be used for pancakes and crackers."))
	triples = append(triples,
		tripleHasTopic(noteURI("starter-maintenance"), topicURI("fermentation")),
	)

	add(note(noteURI("basic-cuts"), "Basic Knife Cuts",
		"Essential knife cuts: julienne (1/8 x 1/8 x 2 inches), brunoise (1/8 inch dice, from julienne), chiffonade (thin ribbons of leafy herbs/greens). Practice with carrots for julienne and brunoise, basil for chiffonade. Keep knife sharp and use claw grip."))
	triples = append(triples,
		tripleHasTopic(noteURI("basic-cuts"), topicURI("knife-skills")),
		tripleTag(noteURI("basic-cuts"), tagURI("essential")),
	)

	add(note(noteURI("flour-types"), "Flour Types and Protein Content",
		"Flour types by protein content: cake flour 7-9%, all-purpose 10-12%, bread flour 12-14%, high-gluten 14-15%. Higher protein means more gluten development. King Arthur AP is on the higher end at 11.7%. Use bread flour for sourdough and pizza."))
	triples = append(triples,
		tripleHasTopic(noteURI("flour-types"), topicURI("ingredients")),
		tripleHasTopic(noteURI("flour-types"), topicURI("baking")),
	)

	// Decisions
	add(decision(decisionURI("sourdough-over-yeast"), "Maintain Sourdough Starter over Commercial Yeast",
		"Chose to maintain a sourdough starter rather than relying on commercial yeast. Sourdough provides superior flavor complexity, better keeping quality, and natural leavening. Worth the daily maintenance effort for the depth of flavor."))
	triples = append(triples,
		tripleHasTopic(decisionURI("sourdough-over-yeast"), topicURI("fermentation")),
	)

	add(decision(decisionURI("whole-spices"), "Whole Spices over Pre-Ground",
		"Switched to buying whole spices and grinding as needed instead of pre-ground. Whole spices have longer shelf life (2-3 years vs 6 months) and significantly better flavor when freshly toasted and ground."))
	triples = append(triples,
		tripleHasTopic(decisionURI("whole-spices"), topicURI("ingredients")),
	)

	add(decision(decisionURI("whetstone-sharpener"), "Whetstone over Electric Sharpener",
		"Chose whetstone sharpening over electric sharpener. Whetstone produces a better edge, removes less metal, and allows control over bevel angle. 1000/6000 grit combination stone is sufficient for home use."))
	triples = append(triples,
		tripleHasTopic(decisionURI("whetstone-sharpener"), topicURI("knife-skills")),
	)

	// Questions
	add(question(questionURI("rye-hydration"), "What is the ideal hydration percentage for a rye sourdough — rye absorbs more water but too much makes it gummy"))
	triples = append(triples,
		tripleHasTopic(questionURI("rye-hydration"), topicURI("baking")),
		tripleHasTopic(questionURI("rye-hydration"), topicURI("fermentation")),
	)

	add(question(questionURI("mascarpone-substitute"), "Can mascarpone be substituted with cream cheese in tiramisu, and what ratio adjustments are needed"))
	triples = append(triples,
		tripleHasTopic(questionURI("mascarpone-substitute"), topicURI("desserts")),
	)

	add(question(questionURI("dutch-oven-size"), "What is the best Dutch oven size for baking a single sourdough loaf — 5 quart vs 7 quart"))
	triples = append(triples,
		tripleHasTopic(questionURI("dutch-oven-size"), topicURI("baking")),
	)

	// Sources
	add(source(sourceURI("tartine-bread"), "Tartine Bread by Chad Robertson",
		"Tartine Bread by Chad Robertson — definitive guide to country-style sourdough. Covers starter maintenance, high-hydration doughs, shaping, and baking in a Dutch oven. The basic country loaf recipe is the gold standard."))
	triples = append(triples,
		tripleHasTopic(sourceURI("tartine-bread"), topicURI("baking")),
		tripleHasTopic(sourceURI("tartine-bread"), topicURI("fermentation")),
	)

	add(source(sourceURI("food-lab"), "The Food Lab by Kenji Lopez-Alt",
		"The Food Lab by J. Kenji Lopez-Alt — science-driven approach to cooking. Excellent chapters on knife skills, meat cookery, and the science behind browning, emulsification, and gluten development."))
	triples = append(triples,
		tripleHasTopic(sourceURI("food-lab"), topicURI("main-courses")),
	)

	add(source(sourceURI("jaffrey-indian-cooking"), "Madhur Jaffrey's Indian Cooking",
		"Madhur Jaffrey's Indian Cooking — comprehensive guide to Indian cuisine with accessible recipes for home cooks. Covers regional variations, spice blending, and essential techniques for curries, biryanis, and chutneys."))
	triples = append(triples,
		tripleHasTopic(sourceURI("jaffrey-indian-cooking"), topicURI("main-courses")),
	)

	// Cross-references
	triples = append(triples,
		tripleRef(noteURI("sourdough-recipe"), entityURI("dutch-oven")),
		tripleRef(noteURI("sourdough-recipe"), sourceURI("tartine-bread")),
		tripleRef(noteURI("flour-types"), entityURI("king-arthur-flour")),
		tripleRef(noteURI("flour-types"), entityURI("bobs-red-mill")),
		tripleRef(noteURI("chicken-tikka-masala"), sourceURI("jaffrey-indian-cooking")),
		tripleRef(questionURI("mascarpone-substitute"), noteURI("tiramisu")),
		tripleRef(questionURI("dutch-oven-size"), entityURI("dutch-oven")),
	)

	return testProject{
		Name:    "recipe-collection",
		Graph:   "project:recipe-collection",
		Triples: triples,
		Queries: []searchQuery{
			{
				Name:       "search sourdough",
				Args:       map[string]any{"query": "sourdough"},
				MinResults: 1,
				MustFind:   []string{noteURI("sourdough-recipe"), noteURI("starter-maintenance"), decisionURI("sourdough-over-yeast")},
			},
			{
				Name:       "search chicken tikka",
				Args:       map[string]any{"query": "chicken tikka"},
				MinResults: 1,
				MustFind:   []string{noteURI("chicken-tikka-masala")},
			},
			{
				Name:       "search tiramisu mascarpone",
				Args:       map[string]any{"query": "tiramisu mascarpone"},
				MinResults: 1,
				MustFind:   []string{noteURI("tiramisu")},
			},
			{
				Name:       "search fermentation starter",
				Args:       map[string]any{"query": "fermentation starter"},
				MinResults: 1,
				MustFind:   []string{noteURI("starter-maintenance"), topicURI("fermentation")},
			},
			{
				Name:       "search knife skills julienne",
				Args:       map[string]any{"query": "knife skills julienne"},
				MinResults: 1,
				MustFind:   []string{noteURI("basic-cuts")},
			},
			{
				Name:       "search flour protein",
				Args:       map[string]any{"query": "flour protein"},
				MinResults: 1,
				MustFind:   []string{noteURI("flour-types")},
			},
			{
				Name:       "search King Arthur",
				Args:       map[string]any{"query": "King Arthur"},
				MinResults: 1,
				MustFind:   []string{entityURI("king-arthur-flour")},
			},
			{
				Name:       "search garam masala turmeric",
				Args:       map[string]any{"query": "garam masala turmeric"},
				MinResults: 1,
				MustFind:   []string{noteURI("chicken-tikka-masala")},
			},
			{
				Name:       "type filter notes",
				Args:       map[string]any{"query": "recipe", "type": testHippoNote},
				MinResults: 1,
				MustNotFind: []string{topicURI("baking"), entityURI("king-arthur-flour")},
			},
			{
				Name:       "search Dutch oven",
				Args:       map[string]any{"query": "Dutch oven"},
				MinResults: 1,
				MustFind:   []string{entityURI("dutch-oven")},
			},
			{
				Name:       "search hydration",
				Args:       map[string]any{"query": "hydration"},
				MinResults: 1,
				MustFind:   []string{noteURI("sourdough-recipe"), questionURI("rye-hydration")},
			},
			{
				Name:       "search Madhur Jaffrey",
				Args:       map[string]any{"query": "Madhur Jaffrey"},
				MinResults: 1,
				MustFind:   []string{entityURI("madhur-jaffrey"), sourceURI("jaffrey-indian-cooking")},
			},
			{
				Name:       "search Tartine bread",
				Args:       map[string]any{"query": "Tartine bread"},
				MinResults: 1,
				MustFind:   []string{sourceURI("tartine-bread")},
			},
			{
				Name:       "search whetstone sharpener",
				Args:       map[string]any{"query": "whetstone sharpener"},
				MinResults: 1,
				MustFind:   []string{decisionURI("whetstone-sharpener")},
			},
			{
				Name:       "search meal planning",
				Args:       map[string]any{"query": "meal planning"},
				MinResults: 1,
				MustFind:   []string{topicURI("meal-planning")},
			},
			{
				Name:       "search Burlap Barrel spices",
				Args:       map[string]any{"query": "Burlap Barrel spices"},
				MinResults: 1,
				MustFind:   []string{entityURI("burlap-and-barrel")},
			},
			{
				Name:       "limit test",
				Args:       map[string]any{"query": "flour", "limit": 2},
				MaxResults: 2,
			},
			{
				Name:       "negative search permit building",
				Args:       map[string]any{"query": "permit building"},
				MaxResults: 0,
			},
			{
				Name:       "case insensitive sourdough",
				Args:       map[string]any{"query": "SOURDOUGH"},
				MinResults: 1,
				MustFind:   []string{noteURI("sourdough-recipe")},
			},
			{
				Name:       "search brunoise chiffonade",
				Args:       map[string]any{"query": "brunoise chiffonade"},
				MinResults: 1,
				MustFind:   []string{noteURI("basic-cuts")},
			},
			{
				Name:       "search cream cheese substitute",
				Args:       map[string]any{"query": "cream cheese substitute"},
				MinResults: 1,
				MustFind:   []string{questionURI("mascarpone-substitute")},
			},
			{
				Name:       "search Bobs Red Mill",
				Args:       map[string]any{"query": "Bob's Red Mill"},
				MinResults: 1,
				MustFind:   []string{entityURI("bobs-red-mill")},
			},
			{
				Name:       "search zaatar paprika",
				Args:       map[string]any{"query": "za'atar paprika"},
				MinResults: 1,
				MustFind:   []string{entityURI("burlap-and-barrel")},
			},
		},
	}
}
