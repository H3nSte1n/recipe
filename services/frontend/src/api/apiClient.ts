import { logout } from '../services/authService';

export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const response = await fetch(input, init);
  if (response.status === 401) {
    logout();
    window.location.href = '/';
    return response;
  }
  return response;
}
