import { apiFetch } from '../api/apiClient';
import { getAuthHeaders } from './authService';
import type { AIConfig, AIModel, Profile, ProfileUpdate } from '../types/profile';

async function request<T>(path: string, method = 'GET', body?: object): Promise<T> {
  const response = await apiFetch(`/api/v1${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) {
    if (response.status === 403) throw new Error('Verify your email before saving changes.');
    if (response.status === 400 || response.status === 409) {
      const detail: unknown = await response.json().catch(() => null);
      if (detail && typeof detail === 'object' && 'error' in detail && typeof detail.error === 'string') {
        throw new Error(detail.error);
      }
    }
    throw new Error(`Could not save or load data (${response.status}). Please try again.`);
  }
  return response.json() as Promise<T>;
}

export const getProfile = () => request<Profile>('/users');
export const updateProfile = (data: ProfileUpdate) => request<Profile>('/users', 'PUT', data);
export const getAIModels = () => request<AIModel[]>('/ai-configs/models');
export const getAIConfigs = () => request<AIConfig[]>('/ai-configs');
export const createAIConfig = (ai_model_id: string, api_key: string, is_default: boolean) =>
  request<AIConfig>('/ai-configs', 'POST', { ai_model_id, api_key, is_default });
export const updateAIConfig = (id: string, data: { ai_model_id?: string; api_key?: string }) =>
  request<AIConfig>(`/ai-configs/${encodeURIComponent(id)}`, 'PUT', data);
export const setDefaultAIConfig = (id: string) =>
  request<{ message: string }>(`/ai-configs/${encodeURIComponent(id)}/set-default`, 'POST');
export const deleteAIConfig = (id: string) =>
  request<{ message: string }>(`/ai-configs/${encodeURIComponent(id)}`, 'DELETE');
