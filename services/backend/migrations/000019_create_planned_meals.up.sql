CREATE UNIQUE INDEX idx_recipes_id_user_id ON recipes(id, user_id);

CREATE TABLE planned_meals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID NOT NULL,
    cook_date DATE NOT NULL,
    meal_type VARCHAR(16) CHECK (meal_type IN ('breakfast', 'lunch', 'dinner', 'snack')),
    servings INTEGER CHECK (servings BETWEEN 1 AND 100),
    notes TEXT CHECK (char_length(notes) <= 1000),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT planned_meals_owned_recipe_fk FOREIGN KEY (recipe_id, user_id)
        REFERENCES recipes(id, user_id) ON DELETE CASCADE ON UPDATE RESTRICT
);

CREATE INDEX idx_planned_meals_user_date_id ON planned_meals(user_id, cook_date, id);
CREATE INDEX idx_planned_meals_recipe_id ON planned_meals(recipe_id);

CREATE TRIGGER update_planned_meals_updated_at
    BEFORE UPDATE ON planned_meals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
