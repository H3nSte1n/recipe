import { apiFetch } from '../api/apiClient';
import { getAuthHeaders } from './authService';
import type { PlannedMeal, PlannedMealInput, PlannedMealPage } from '../types/plannedMeal';

export class PlannedMealApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'PlannedMealApiError';
  }
}

// The backend only sends fixed, client-safe text with a 400 (e.g. "only published recipes can be
// planned"), so that is shown to explain what to fix. Other statuses get a message of our own.
function friendlyMessage(status: number, serverMessage?: string): string {
  if (status === 400) {
    return serverMessage
      ? `Those meal details are not valid: ${serverMessage}.`
      : 'Those meal details are not valid. Please check them and try again.';
  }
  if (status === 403) return 'Please verify your email before changing your meal plan.';
  if (status === 404) return 'That planned meal or recipe no longer exists.';
  if (status >= 500) return 'Something went wrong on our side. Please try again.';
  return 'Could not complete the request. Please try again.';
}

async function serverMessage(response: Response): Promise<string | undefined> {
  try {
    const body: unknown = await response.json();
    const message = typeof body === 'object' && body !== null ? (body as { error?: unknown }).error : undefined;
    return typeof message === 'string' && message.length <= 200 ? message : undefined;
  } catch {
    return undefined;
  }
}

async function checked(response: Response): Promise<Response> {
  if (!response.ok) {
    const detail = response.status === 400 ? await serverMessage(response) : undefined;
    throw new PlannedMealApiError(response.status, friendlyMessage(response.status, detail));
  }
  return response;
}

// Network failures and malformed responses carry browser text ("Failed to fetch"), so only errors
// this service produced are shown to the user.
export function plannedMealErrorMessage(failure: unknown, fallback: string): string {
  return failure instanceof PlannedMealApiError ? failure.message : fallback;
}

function headers(): HeadersInit {
  return { 'Content-Type': 'application/json', ...getAuthHeaders() };
}

export async function listPlannedMeals(from: string, to: string, page: number, signal?: AbortSignal): Promise<PlannedMealPage> {
  const query = new URLSearchParams({ from, to, page: String(page), page_size: '100' });
  const response = await checked(await apiFetch(`/api/v1/planned-meals?${query}`, {
    headers: headers(), signal,
  }));
  return response.json() as Promise<PlannedMealPage>;
}

export async function listAllPlannedMeals(from: string, to: string, signal?: AbortSignal): Promise<PlannedMeal[]> {
  const items: PlannedMeal[] = [];
  let page = 1;
  for (;;) {
    const result = await listPlannedMeals(from, to, page, signal);
    items.push(...result.items);
    if (items.length >= result.total || result.items.length === 0) return items;
    page += 1;
  }
}

export async function createPlannedMeal(input: PlannedMealInput): Promise<PlannedMeal> {
  const response = await checked(await apiFetch('/api/v1/planned-meals', {
    method: 'POST', headers: headers(), body: JSON.stringify(input),
  }));
  return response.json() as Promise<PlannedMeal>;
}

export async function updatePlannedMeal(id: string, input: PlannedMealInput): Promise<PlannedMeal> {
  const response = await checked(await apiFetch(`/api/v1/planned-meals/${encodeURIComponent(id)}`, {
    method: 'PUT', headers: headers(), body: JSON.stringify(input),
  }));
  return response.json() as Promise<PlannedMeal>;
}

export async function deletePlannedMeal(id: string): Promise<void> {
  await checked(await apiFetch(`/api/v1/planned-meals/${encodeURIComponent(id)}`, {
    method: 'DELETE', headers: headers(),
  }));
}
