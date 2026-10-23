import { apiFetch } from '../api/apiClient';
import { getAuthHeaders } from './authService';
import type { PlannedMeal, PlannedMealInput, PlannedMealPage } from '../types/plannedMeal';

export class PlannedMealApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'PlannedMealApiError';
  }
}

// Server messages name fields and limits, so users get a message per status instead.
function friendlyMessage(status: number): string {
  if (status === 400) return 'Those meal details are not valid. Please check them and try again.';
  if (status === 403) return 'Please verify your email before changing your meal plan.';
  if (status === 404) return 'That planned meal or recipe no longer exists.';
  if (status >= 500) return 'Something went wrong on our side. Please try again.';
  return 'Could not complete the request. Please try again.';
}

async function checked(response: Response): Promise<Response> {
  if (!response.ok) throw new PlannedMealApiError(response.status, friendlyMessage(response.status));
  return response;
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
