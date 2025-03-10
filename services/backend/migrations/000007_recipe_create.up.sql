CREATE TABLE recipes (
                         id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                         user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                         title VARCHAR(255) NOT NULL,
                         description TEXT,
                         notes TEXT,
                         rating DECIMAL DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
                         image_url VARCHAR(255),
                         source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('URL', 'MANUAL', 'PDF', 'IMAGE')),
                         source TEXT,
                         is_private BOOLEAN DEFAULT false,
                         servings INTEGER NOT NULL CHECK (servings > 0),
                         prep_time INTEGER CHECK (prep_time >= 0),
                         cook_time INTEGER CHECK (cook_time >= 0),
                         status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
                         created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recipe_ingredients (
                                    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
                                    name VARCHAR(255) NOT NULL,
                                    amount DECIMAL CHECK (amount >= 0),
                                    unit VARCHAR(50),
                                    notes TEXT,
                                    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recipe_instructions (
                                     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                     recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
                                     step_number INTEGER NOT NULL CHECK (step_number > 0),
                                     instruction TEXT NOT NULL,
                                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                     updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recipe_nutrition (
                                  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                  recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    -- Base nutrition
                                  calories DECIMAL CHECK (calories >= 0),
                                  per_serving BOOLEAN DEFAULT true,
    -- Macro nutrition
                                  protein DECIMAL CHECK (protein >= 0),
                                  carbs DECIMAL CHECK (carbs >= 0),
                                  fat DECIMAL CHECK (fat >= 0),
                                  fiber DECIMAL CHECK (fiber >= 0),
                                  sugar DECIMAL CHECK (sugar >= 0),
                                  saturated_fat DECIMAL CHECK (saturated_fat >= 0),
                                  cholesterol DECIMAL CHECK (cholesterol >= 0),
                                  sodium DECIMAL CHECK (sodium >= 0),
    -- Micro nutrition
                                  vitamin_a DECIMAL CHECK (vitamin_a >= 0),    -- IU
                                  vitamin_c DECIMAL CHECK (vitamin_c >= 0),    -- mg
                                  vitamin_d DECIMAL CHECK (vitamin_d >= 0),    -- IU
                                  vitamin_e DECIMAL CHECK (vitamin_e >= 0),    -- mg
                                  vitamin_k DECIMAL CHECK (vitamin_k >= 0),    -- mcg
                                  thiamin DECIMAL CHECK (thiamin >= 0),        -- mg
                                  riboflavin DECIMAL CHECK (riboflavin >= 0),  -- mg
                                  niacin DECIMAL CHECK (niacin >= 0),          -- mg
                                  vitamin_b6 DECIMAL CHECK (vitamin_b6 >= 0),  -- mg
                                  vitamin_b12 DECIMAL CHECK (vitamin_b12 >= 0),-- mcg
                                  folate DECIMAL CHECK (folate >= 0),          -- mcg
                                  calcium DECIMAL CHECK (calcium >= 0),        -- mg
                                  iron DECIMAL CHECK (iron >= 0),              -- mg
                                  magnesium DECIMAL CHECK (magnesium >= 0),    -- mg
                                  phosphorus DECIMAL CHECK (phosphorus >= 0),   -- mg
                                  potassium DECIMAL CHECK (potassium >= 0),    -- mg
                                  zinc DECIMAL CHECK (zinc >= 0),              -- mg
                                  selenium DECIMAL CHECK (selenium >= 0),       -- mcg
                                  copper DECIMAL CHECK (copper >= 0),          -- mg
                                  manganese DECIMAL CHECK (manganese >= 0),     -- mg
                                  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sub_recipes (
                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                             parent_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
                             child_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
                             serving_factor DECIMAL DEFAULT 1 CHECK (serving_factor >= 0.1),
                             created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                             updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                             UNIQUE(parent_id, child_id)
);

-- Create indexes
CREATE INDEX idx_recipes_user_id ON recipes(user_id);
CREATE INDEX idx_recipes_is_private ON recipes(is_private);
CREATE INDEX idx_recipe_ingredients_recipe_id ON recipe_ingredients(recipe_id);
CREATE INDEX idx_recipe_instructions_recipe_id ON recipe_instructions(recipe_id);
CREATE INDEX idx_recipe_nutrition_recipe_id ON recipe_nutrition(recipe_id);
CREATE INDEX idx_sub_recipes_parent_id ON sub_recipes(parent_id);
CREATE INDEX idx_sub_recipes_child_id ON sub_recipes(child_id);
