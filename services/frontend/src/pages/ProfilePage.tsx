import { useEffect, useState, type FormEvent } from 'react';
import {
  createAIConfig, deleteAIConfig, getAIConfigs, getAIModels, getProfile,
  setDefaultAIConfig, updateAIConfig, updateProfile,
} from '../services/profileService';
import type { AIConfig, AIModel, ProfileUpdate } from '../types/profile';
import '../styles/ProfilePage.css';

interface ProfilePageProps {
  onBack: () => void;
  onLogout: () => void;
}

const emptyProfile: ProfileUpdate = {
  first_name: '', last_name: '', bio: '', location: '', website_url: '',
};

function modelLabel(model: AIModel): string {
  return `${model.provider} · ${model.name}`;
}

export default function ProfilePage({ onBack, onLogout }: ProfilePageProps) {
  const [profile, setProfile] = useState<ProfileUpdate>(emptyProfile);
  const [email, setEmail] = useState('');
  const [models, setModels] = useState<AIModel[]>([]);
  const [configs, setConfigs] = useState<AIConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [aiLoadError, setAILoadError] = useState('');
  const [profileError, setProfileError] = useState('');
  const [profileMessage, setProfileMessage] = useState('');
  const [profileSaving, setProfileSaving] = useState(false);
  const [newModelId, setNewModelId] = useState('');
  const [newKey, setNewKey] = useState('');
  const [newError, setNewError] = useState('');
  const [adding, setAdding] = useState(false);
  const [busyConfigId, setBusyConfigId] = useState<string | null>(null);
  const [configError, setConfigError] = useState<Record<string, string>>({});
  const [configMessage, setConfigMessage] = useState<Record<string, string>>({});
  const [draftModels, setDraftModels] = useState<Record<string, string>>({});
  const [draftKeys, setDraftKeys] = useState<Record<string, string>>({});
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    async function load() {
      try {
        const [profileResult, modelsResult, configsResult] = await Promise.allSettled([
          getProfile(), getAIModels(), getAIConfigs(),
        ]);
        if (!active) return;
        if (profileResult.status === 'fulfilled') {
          const data = profileResult.value;
          setProfile({
            first_name: data.user?.first_name ?? '',
            last_name: data.user?.last_name ?? '',
            bio: data.bio ?? '',
            location: data.location ?? '',
            website_url: data.website_url ?? '',
          });
          setEmail(data.user?.email ?? '');
        } else {
          setLoadError(profileResult.reason instanceof Error ? profileResult.reason.message : 'Could not load profile.');
        }
        if (modelsResult.status === 'fulfilled' && configsResult.status === 'fulfilled') {
          setModels(modelsResult.value ?? []);
          setConfigs(configsResult.value ?? []);
          setNewModelId(modelsResult.value?.[0]?.id ?? '');
        } else {
          setAILoadError('Could not load AI settings. Refresh the page to try again.');
        }
      } catch (error) {
        if (active) setLoadError(error instanceof Error ? error.message : 'Could not load profile.');
      } finally {
        if (active) setLoading(false);
      }
    }
    void load();
    return () => { active = false; };
  }, []);

  const updateField = (field: keyof ProfileUpdate, value: string) => {
    setProfile(current => ({ ...current, [field]: value }));
    setProfileMessage('');
  };

  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProfileError('');
    setProfileMessage('');
    if (!profile.first_name.trim() || !profile.last_name.trim()) {
      setProfileError('First and last name are required.');
      return;
    }
    setProfileSaving(true);
    try {
      const saved = await updateProfile({
        ...profile,
        first_name: profile.first_name.trim(),
        last_name: profile.last_name.trim(),
        website_url: profile.website_url.trim(),
      });
      setProfile({
        first_name: saved.user.first_name,
        last_name: saved.user.last_name,
        bio: saved.bio ?? '',
        location: saved.location ?? '',
        website_url: saved.website_url ?? '',
      });
      setProfileMessage('Profile saved.');
    } catch (error) {
      setProfileError(error instanceof Error ? error.message : 'Could not save profile.');
    } finally {
      setProfileSaving(false);
    }
  }

  async function refreshConfigs() {
    setConfigs(await getAIConfigs());
  }

  async function addConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNewError('');
    const modelId = availableNewModels.some(model => model.id === newModelId) ? newModelId : availableNewModels[0]?.id;
    if (!modelId || !newKey.trim()) {
      setNewError('Choose a model and enter its API token.');
      return;
    }
    setAdding(true);
    try {
      await createAIConfig(modelId, newKey.trim(), configs.length === 0);
      await refreshConfigs();
      setNewKey('');
    } catch (error) {
      setNewError(error instanceof Error ? error.message : 'Could not add model.');
    } finally {
      setAdding(false);
    }
  }

  async function saveConfig(config: AIConfig) {
    const modelId = draftModels[config.id] ?? config.ai_model_id;
    const key = draftKeys[config.id]?.trim() ?? '';
    if (modelId === config.ai_model_id && !key) return;
    const oldProvider = config.ai_model?.provider ?? models.find(model => model.id === config.ai_model_id)?.provider;
    const newProvider = models.find(model => model.id === modelId)?.provider;
    if (oldProvider && newProvider && oldProvider !== newProvider && !key) {
      setConfigError(current => ({ ...current, [config.id]: 'Enter a token for the new provider.' }));
      return;
    }
    setConfigError(current => ({ ...current, [config.id]: '' }));
    setConfigMessage(current => ({ ...current, [config.id]: '' }));
    setBusyConfigId(config.id);
    try {
      await updateAIConfig(config.id, {
        ...(modelId !== config.ai_model_id ? { ai_model_id: modelId } : {}),
        ...(key ? { api_key: key } : {}),
      });
      await refreshConfigs();
      setDraftKeys(current => ({ ...current, [config.id]: '' }));
      setConfigMessage(current => ({ ...current, [config.id]: 'Model settings saved.' }));
    } catch (error) {
      setConfigError(current => ({ ...current, [config.id]: error instanceof Error ? error.message : 'Could not save model.' }));
    } finally {
      setBusyConfigId(null);
    }
  }

  async function makeDefault(config: AIConfig) {
    setBusyConfigId(config.id);
    setConfigError(current => ({ ...current, [config.id]: '' }));
    try {
      await setDefaultAIConfig(config.id);
      await refreshConfigs();
    } catch (error) {
      setConfigError(current => ({ ...current, [config.id]: error instanceof Error ? error.message : 'Could not set default model.' }));
    } finally {
      setBusyConfigId(null);
    }
  }

  async function removeConfig(config: AIConfig) {
    setBusyConfigId(config.id);
    setConfigError(current => ({ ...current, [config.id]: '' }));
    try {
      await deleteAIConfig(config.id);
      await refreshConfigs();
      setConfirmDeleteId(null);
    } catch (error) {
      setConfigError(current => ({ ...current, [config.id]: error instanceof Error ? error.message : 'Could not remove model.' }));
    } finally {
      setBusyConfigId(null);
    }
  }

  const availableNewModels = models.filter(model => !configs.some(config => config.ai_model_id === model.id));

  return (
    <div className="profile-page">
      <header className="profile-page__topbar">
        <button className="profile-page__back" type="button" onClick={onBack} aria-label="Back to recipes">
          <span aria-hidden="true">←</span> Recipes
        </button>
        <span className="profile-page__brand">Mise</span>
        <button className="profile-page__signout" type="button" onClick={onLogout}>Sign out</button>
      </header>

      <main className="profile-page__main">
        <div className="profile-page__intro">
          <span className="profile-page__eyebrow">YOUR ACCOUNT</span>
          <h1>My profile</h1>
          <p>Keep your details and cooking assistant ready for your next recipe.</p>
        </div>

        {loading && <p className="profile-page__status" role="status">Loading profile…</p>}
        {loadError && <p className="profile-page__error" role="alert">{loadError}</p>}

        {!loading && !loadError && (
          <div className="profile-page__sections">
            <section className="profile-page__section" aria-labelledby="profile-details-title">
              <div className="profile-page__section-heading">
                <div>
                  <span className="profile-page__eyebrow">THE PERSON BEHIND THE RECIPES</span>
                  <h2 id="profile-details-title">About you</h2>
                  <p>The details shown on your profile.</p>
                </div>
              </div>
              <form className="profile-page__form" onSubmit={event => void saveProfile(event)}>
                <div className="profile-page__row">
                  <label>First name<input value={profile.first_name} maxLength={100} autoComplete="given-name" onChange={event => updateField('first_name', event.target.value)} required /></label>
                  <label>Last name<input value={profile.last_name} maxLength={100} autoComplete="family-name" onChange={event => updateField('last_name', event.target.value)} required /></label>
                </div>
                <label>Email address<input value={email} type="email" autoComplete="email" readOnly aria-describedby="profile-email-note" /></label>
                <p id="profile-email-note" className="profile-page__hint">Your email address cannot be changed here.</p>
                <div className="profile-page__row">
                  <label>Location<input value={profile.location} maxLength={255} autoComplete="address-level2" placeholder="Where do you cook?" onChange={event => updateField('location', event.target.value)} /></label>
                  <label>Website<input value={profile.website_url} maxLength={255} type="url" placeholder="https://example.com" onChange={event => updateField('website_url', event.target.value)} /></label>
                </div>
                <label>Bio<textarea value={profile.bio} rows={4} placeholder="A few words about you and what you love to cook" onChange={event => updateField('bio', event.target.value)} /></label>
                <div className="profile-page__form-footer">
                  <div aria-live="polite">{profileError && <p className="profile-page__error">{profileError}</p>}{profileMessage && <p className="profile-page__success">{profileMessage}</p>}</div>
                  <button className="profile-page__primary" type="submit" disabled={profileSaving}>{profileSaving ? 'Saving…' : 'Save profile'}</button>
                </div>
              </form>
            </section>

            <section className="profile-page__section" aria-labelledby="profile-ai-title">
              <div className="profile-page__section-heading">
                <div>
                  <span className="profile-page__eyebrow">YOUR COOKING ASSISTANT</span>
                  <h2 id="profile-ai-title">AI models</h2>
                  <p>Choose a model and keep its API token up to date.</p>
                </div>
              </div>
              <div className="profile-page__ai-content">
                {aiLoadError && <p className="profile-page__error" role="alert">{aiLoadError}</p>}
                {!aiLoadError && <>
                {configs.length === 0 && <p className="profile-page__hint">No AI model configured yet. Add one to use AI recipe features.</p>}
                {configs.map(config => {
                  const selectedModel = draftModels[config.id] ?? config.ai_model_id;
                  const hasChanges = selectedModel !== config.ai_model_id || Boolean(draftKeys[config.id]?.trim());
                  const selectableModels = models.filter(model => model.id === config.ai_model_id || !configs.some(other => other.id !== config.id && other.ai_model_id === model.id));
                  if (config.ai_model && !selectableModels.some(model => model.id === config.ai_model_id)) selectableModels.unshift(config.ai_model);
                  return (
                    <div className="profile-page__config" key={config.id}>
                      <div className="profile-page__config-heading">
                        <div>
                          <span className="profile-page__eyebrow">{config.ai_model?.provider ?? 'AI MODEL'}</span>
                          <h3>{config.ai_model?.name ?? models.find(model => model.id === config.ai_model_id)?.name ?? 'Model'}</h3>
                        </div>
                        {config.is_default && <span className="profile-page__badge">Default model</span>}
                      </div>
                      <div className="profile-page__row">
                        <label>Model
                          <select value={selectedModel} onChange={event => { setDraftModels(current => ({ ...current, [config.id]: event.target.value })); setConfigMessage(current => ({ ...current, [config.id]: '' })); }}>
                            {selectableModels.map(model => <option key={model.id} value={model.id}>{modelLabel(model)}{model.is_active ? '' : ' (unavailable)'}</option>)}
                          </select>
                        </label>
                        <label>API token
                          <input type="password" autoComplete="new-password" value={draftKeys[config.id] ?? ''} placeholder="••••••••  ·  Token saved" onChange={event => { setDraftKeys(current => ({ ...current, [config.id]: event.target.value })); setConfigMessage(current => ({ ...current, [config.id]: '' })); }} />
                        </label>
                      </div>
                      <p className="profile-page__hint">Leave the token blank to keep the current one.</p>
                      <div className="profile-page__config-actions">
                        {!config.is_default && <button className="profile-page__text-button" type="button" disabled={busyConfigId !== null} onClick={() => void makeDefault(config)}>Set as default</button>}
                        {confirmDeleteId === config.id ? (
                          <span className="profile-page__confirm">
                            Remove this model?
                            <button type="button" disabled={busyConfigId !== null} onClick={() => void removeConfig(config)}>Remove</button>
                            <button type="button" onClick={() => setConfirmDeleteId(null)}>Cancel</button>
                          </span>
                        ) : <button className="profile-page__text-button profile-page__text-button--danger" type="button" onClick={() => setConfirmDeleteId(config.id)}>Remove</button>}
                        <button className="profile-page__primary profile-page__primary--small" type="button" disabled={!hasChanges || busyConfigId !== null} onClick={() => void saveConfig(config)}>{busyConfigId === config.id ? 'Saving…' : 'Save changes'}</button>
                      </div>
                      <div aria-live="polite">{configError[config.id] && <p className="profile-page__error">{configError[config.id]}</p>}{configMessage[config.id] && <p className="profile-page__success">{configMessage[config.id]}</p>}</div>
                    </div>
                  );
                })}
                {availableNewModels.length > 0 && (
                  <form className="profile-page__add-model" onSubmit={event => void addConfig(event)}>
                    <h3>Add a model</h3>
                    <div className="profile-page__row">
                      <label>Model
                        <select value={availableNewModels.some(model => model.id === newModelId) ? newModelId : availableNewModels[0].id} onChange={event => setNewModelId(event.target.value)}>
                          {availableNewModels.map(model => <option key={model.id} value={model.id}>{modelLabel(model)}</option>)}
                        </select>
                      </label>
                      <label>API token
                        <input type="password" autoComplete="new-password" value={newKey} placeholder="Enter your API token" onChange={event => setNewKey(event.target.value)} required />
                      </label>
                    </div>
                    <div className="profile-page__form-footer">
                      <div aria-live="polite">{newError && <p className="profile-page__error">{newError}</p>}</div>
                      <button className="profile-page__primary" type="submit" disabled={adding}>{adding ? 'Adding…' : 'Add model'}</button>
                    </div>
                  </form>
                )}
                </>}
              </div>
            </section>
          </div>
        )}
      </main>
    </div>
  );
}
