export interface Profile {
  id: string;
  bio: string;
  location: string;
  website_url: string;
  user: {
    first_name: string;
    last_name: string;
    email: string;
    email_verified_at: string | null;
  };
}

export interface ProfileUpdate {
  first_name: string;
  last_name: string;
  bio: string;
  location: string;
  website_url: string;
}

export interface AIModel {
  id: string;
  name: string;
  provider: string;
  model_version: string;
  is_active: boolean;
}

export interface AIConfig {
  id: string;
  ai_model_id: string;
  is_default: boolean;
  ai_model?: AIModel;
}
