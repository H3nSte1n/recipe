-- Seed data for common store chains in the Netherlands and Germany

-- Albert Heijn (Netherlands)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Albert Heijn',
    'NL',
    '[
        {"order": 1, "name": "Fruit & Groenten", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Bakkerij", "categories": ["BAKERY"]},
        {"order": 3, "name": "Vlees & Vis", "categories": ["MEAT"]},
        {"order": 4, "name": "Zuivel & Eieren", "categories": ["DAIRY"]},
        {"order": 5, "name": "Diepvries", "categories": ["FROZEN"]},
        {"order": 6, "name": "Dranken", "categories": ["BEVERAGES"]},
        {"order": 7, "name": "Voorraadkast", "categories": ["PANTRY"]},
        {"order": 8, "name": "Huishouden", "categories": ["HOUSEHOLD"]},
        {"order": 9, "name": "Overig", "categories": ["OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);

-- Jumbo (Netherlands)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'Jumbo',
    'NL',
    '[
        {"order": 1, "name": "Groente & Fruit", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Brood & Gebak", "categories": ["BAKERY"]},
        {"order": 3, "name": "Vlees, Vis & Vegetarisch", "categories": ["MEAT"]},
        {"order": 4, "name": "Zuivel, Eieren & Boter", "categories": ["DAIRY"]},
        {"order": 5, "name": "Diepvries", "categories": ["FROZEN"]},
        {"order": 6, "name": "Frisdrank & Sappen", "categories": ["BEVERAGES"]},
        {"order": 7, "name": "Boodschappen", "categories": ["PANTRY"]},
        {"order": 8, "name": "Drogisterij", "categories": ["HOUSEHOLD"]},
        {"order": 9, "name": "Overige", "categories": ["OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);

-- Lidl (Netherlands & Germany)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'Lidl',
    'NL',
    '[
        {"order": 1, "name": "Groente & Fruit", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Bakkerij", "categories": ["BAKERY"]},
        {"order": 3, "name": "Koeling", "categories": ["DAIRY", "MEAT"]},
        {"order": 4, "name": "Diepvries", "categories": ["FROZEN"]},
        {"order": 5, "name": "Dranken", "categories": ["BEVERAGES"]},
        {"order": 6, "name": "Droge Producten", "categories": ["PANTRY"]},
        {"order": 7, "name": "Non-Food", "categories": ["HOUSEHOLD", "OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);

-- Aldi (Netherlands & Germany)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000004',
    'Aldi',
    'NL',
    '[
        {"order": 1, "name": "Groente & Fruit", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Vers", "categories": ["BAKERY", "MEAT", "DAIRY"]},
        {"order": 3, "name": "Diepvries", "categories": ["FROZEN"]},
        {"order": 4, "name": "Dranken", "categories": ["BEVERAGES"]},
        {"order": 5, "name": "Overige", "categories": ["PANTRY", "HOUSEHOLD", "OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);

-- Rewe (Germany)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000005',
    'Rewe',
    'DE',
    '[
        {"order": 1, "name": "Obst & Gemüse", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Bäckerei", "categories": ["BAKERY"]},
        {"order": 3, "name": "Fleisch & Wurst", "categories": ["MEAT"]},
        {"order": 4, "name": "Molkereiprodukte", "categories": ["DAIRY"]},
        {"order": 5, "name": "Tiefkühl", "categories": ["FROZEN"]},
        {"order": 6, "name": "Getränke", "categories": ["BEVERAGES"]},
        {"order": 7, "name": "Vorratskammer", "categories": ["PANTRY"]},
        {"order": 8, "name": "Haushalt", "categories": ["HOUSEHOLD"]},
        {"order": 9, "name": "Sonstiges", "categories": ["OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);

-- Edeka (Germany)
INSERT INTO store_chains (id, name, country, layout, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000006',
    'Edeka',
    'DE',
    '[
        {"order": 1, "name": "Obst & Gemüse", "categories": ["PRODUCE"]},
        {"order": 2, "name": "Backwaren", "categories": ["BAKERY"]},
        {"order": 3, "name": "Fleisch & Fisch", "categories": ["MEAT"]},
        {"order": 4, "name": "Milchprodukte", "categories": ["DAIRY"]},
        {"order": 5, "name": "Tiefkühlkost", "categories": ["FROZEN"]},
        {"order": 6, "name": "Getränke", "categories": ["BEVERAGES"]},
        {"order": 7, "name": "Trockenwaren", "categories": ["PANTRY"]},
        {"order": 8, "name": "Drogerie", "categories": ["HOUSEHOLD"]},
        {"order": 9, "name": "Verschiedenes", "categories": ["OTHER"]}
    ]'::jsonb,
    NOW(),
    NOW()
);
