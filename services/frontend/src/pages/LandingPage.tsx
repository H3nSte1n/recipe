import { useState, useEffect, useRef } from 'react';
import { useAuth } from '../hooks/useAuth';
import { register as registerUser } from '../services/authService';
import ScatteredBackground from '../components/ScatteredBackground';
import TunnelControls from '../components/TunnelControls';
import ThemeExplorer from '../components/ThemeExplorer'; // TEMPORARY
import { createDefaultTunnelParams, type TunnelParams } from '../types/tunnelParams';
import '../styles/LandingPage.css';

type AuthView = 'landing' | 'login' | 'register';

interface LandingPageProps {
  onLogin: () => void;
  onRegister: () => void;
}

function HeroView({ onLogin, onRegister }: { onLogin: () => void; onRegister: () => void }) {
  return (
    <>
      <p className="landing-page__brand">Mise</p>
      <h1 className="landing-page__headline type-h1">
        Your recipes,<br />always with you.
      </h1>
      <div className="landing-page__actions">
        <button className="landing-page__btn landing-page__btn--primary" onClick={onLogin}>
          Login
        </button>
        <button className="landing-page__btn landing-page__btn--secondary" onClick={onRegister}>
          Register
        </button>
      </div>
    </>
  );
}

function LoginView({ onSuccess, onBack, onRegister }: { onSuccess: () => void; onBack: () => void; onRegister: () => void }) {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(email, password);
      onSuccess();
    } catch {
      setError('Invalid email or password. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="landing-page__form-wrap">
      <button className="landing-page__back" onClick={onBack} type="button" aria-label="Back">←</button>
      <p className="landing-page__form-title">Login</p>
      <form className="landing-page__form" onSubmit={handleSubmit}>
        <input className="landing-page__input" type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required autoComplete="email" />
        <input className="landing-page__input" type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} required autoComplete="current-password" />
        {error && <p className="landing-page__form-error">{error}</p>}
        <button className="landing-page__submit" type="submit" disabled={loading}>
          {loading ? 'Signing in…' : 'Continue'}
        </button>
      </form>
      <p className="landing-page__form-link" onClick={onRegister}>Create an account</p>
    </div>
  );
}

function RegisterView({ onSuccess, onBack, onLogin }: { onSuccess: () => void; onBack: () => void; onLogin: () => void }) {
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await registerUser(firstName, lastName, email, password);
      onSuccess();
    } catch {
      setError('Registration failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="landing-page__form-wrap">
      <button className="landing-page__back" onClick={onBack} type="button" aria-label="Back">←</button>
      <p className="landing-page__form-title">Create account</p>
      <form className="landing-page__form" onSubmit={handleSubmit}>
        <input className="landing-page__input" type="text" placeholder="First name" value={firstName} onChange={e => setFirstName(e.target.value)} required autoComplete="given-name" />
        <input className="landing-page__input" type="text" placeholder="Last name" value={lastName} onChange={e => setLastName(e.target.value)} required autoComplete="family-name" />
        <input className="landing-page__input" type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required autoComplete="email" />
        <input className="landing-page__input" type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} required autoComplete="new-password" />
        {error && <p className="landing-page__form-error">{error}</p>}
        <button className="landing-page__submit" type="submit" disabled={loading}>
          {loading ? 'Creating account…' : 'Create account'}
        </button>
      </form>
      <p className="landing-page__form-link" onClick={onLogin}>Already have an account? Sign in</p>
    </div>
  );
}

export default function LandingPage({ onLogin, onRegister }: LandingPageProps) {
  const [view, setView] = useState<AuthView>('landing');
  const centerRef = useRef<HTMLDivElement>(null);
  const tunnelParamsRef = useRef<TunnelParams>(createDefaultTunnelParams());

  const switchView = (v: AuthView) => {
    tunnelParamsRef.current.focusMode = v !== 'landing';
    setView(v);
  };

  useEffect(() => {
    // Apply initial blur CSS vars from params
    const el = centerRef.current;
    if (el) {
      const p = tunnelParamsRef.current;
      el.style.setProperty('--blur-padding-x', `${p.blurPaddingX}px`);
      el.style.setProperty('--blur-padding-y', `${p.blurPaddingY}px`);
      el.style.setProperty('--blur-amount', `${p.blurAmount}px`);
    }
  }, []);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!centerRef.current) return;
      const dx = (e.clientX - window.innerWidth / 2) / window.innerWidth * 24;
      const dy = (e.clientY - window.innerHeight / 2) / window.innerHeight * 24;
      centerRef.current.style.transform = `translate(${dx}px, ${dy}px)`;
    };
    window.addEventListener('mousemove', onMouseMove);
    return () => window.removeEventListener('mousemove', onMouseMove);
  }, []);

  return (
    <div className="landing-page">
      <ScatteredBackground paramsRef={tunnelParamsRef} />
      <TunnelControls paramsRef={tunnelParamsRef} blurTargetRef={centerRef} />
      <ThemeExplorer /> {/* TEMPORARY */}
      <div ref={centerRef} className="landing-page__center">
        {view === 'landing' && (
          <HeroView onLogin={() => switchView('login')} onRegister={() => switchView('register')} />
        )}
        {view === 'login' && (
          <LoginView onSuccess={onLogin} onBack={() => switchView('landing')} onRegister={() => switchView('register')} />
        )}
        {view === 'register' && (
          <RegisterView onSuccess={onRegister} onBack={() => switchView('landing')} onLogin={() => switchView('login')} />
        )}
      </div>
    </div>
  );
}
