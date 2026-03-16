const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
const AUTH_TOKEN_KEY = "recipe_auth_token";

export interface RecipeNutrition {
  calories: number;
  per_serving: boolean;
  protein: number;
  carbs: number;
  fat: number;
  fiber: number;
}

export interface Recipe {
  id: string;
  user_id: string;
  title: string;
  description: string;
  notes: string;
  rating: number;
  image_url?: string;
  source_type: string;
  is_private: boolean;
  servings: number;
  prep_time: number;
  cook_time: number;
  shelf_life: number;
  status: string;
  created_at: string;
  updated_at: string;
  nutrition?: RecipeNutrition;
}

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    created_at: string;
    updated_at: string;
  };
}

export interface PublicRecipesResponse {
  recipes: Recipe[];
  total: number;
  page: number;
  size: number;
}

interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  requiresAuth?: boolean;
}

function getAuthToken(): string | null {
  return localStorage.getItem(AUTH_TOKEN_KEY);
}

function setAuthToken(token: string): void {
  localStorage.setItem(AUTH_TOKEN_KEY, token);
}

function clearAuthToken(): void {
  localStorage.removeItem(AUTH_TOKEN_KEY);
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getAuthToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  if (options.requiresAuth && !token) {
    throw new Error("Not authenticated");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: options.method ?? "GET",
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  const raw = await response.text();
  const data = raw ? JSON.parse(raw) : null;

  if (!response.ok) {
    if (response.status === 401) {
      clearAuthToken();
    }
    const message =
      (data && typeof data.error === "string" && data.error) ||
      `Request failed (${response.status})`;
    throw new Error(message);
  }

  return data as T;
}

export const authApi = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    });

    setAuthToken(response.token);
    return response;
  },
  logout(): void {
    clearAuthToken();
  },
  isAuthenticated(): boolean {
    return Boolean(getAuthToken());
  },
  getToken(): string | null {
    return getAuthToken();
  },
};

export const recipeApi = {
  async getRecipes(): Promise<Recipe[]> {
    return request<Recipe[]>("/api/v1/recipes", { requiresAuth: true });
  },
};

export function getApiBaseUrl(): string {
  return API_BASE_URL;
}