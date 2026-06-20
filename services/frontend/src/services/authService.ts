import { AuthResponse, LoginRequest } from '../types/auth';

const TOKEN_KEY = 'token';

export async function login(email: string, password: string): Promise<void> {
  const body: LoginRequest = { email, password };

  try {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      throw new Error(`Login failed: ${response.status} ${response.statusText}`);
    }

    const data: AuthResponse = await response.json();
    localStorage.setItem(TOKEN_KEY, data.token);
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected error occurred during login');
  }
}

export function logout(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function isTokenExpired(): boolean {
  const token = getToken();
  if (!token) {
    return true;
  }
  try {
    const payload = token.split('.')[1];
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    const { exp } = JSON.parse(decoded) as { exp: number };
    return Date.now() / 1000 >= exp;
  } catch {
    return true;
  }
}

export function isAuthenticated(): boolean {
  return !!getToken() && !isTokenExpired();
}

export function getAuthHeaders(): { Authorization: string } | Record<string, never> {
  const token = getToken();
  if (!token) {
    return {};
  }
  return { Authorization: `Bearer ${token}` };
}
