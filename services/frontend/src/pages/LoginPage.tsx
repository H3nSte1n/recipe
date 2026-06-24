import { useState } from 'react';
import { useAuth } from '../hooks/useAuth';
import ScatteredBackground from '../components/ScatteredBackground';
import '../styles/LoginPage.css';

interface LoginPageProps {
  onLogin: () => void;
  onBack?: () => void;
}

export default function LoginPage({ onLogin, onBack }: LoginPageProps) {
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
      onLogin();
    } catch {
      setError('Invalid email or password. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <ScatteredBackground />
      <div className="login-page__form-area">
        {onBack && (
          <button className="login-page__back" onClick={onBack} type="button" aria-label="Back">
            ←
          </button>
        )}
        <p className="login-page__label">Login</p>
        <form className="login-page__form" onSubmit={handleSubmit}>
          <input
            className="login-page__input"
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
          <input
            className="login-page__input"
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
          {error && <p className="login-page__error">{error}</p>}
          <button className="login-page__submit" type="submit" disabled={loading}>
            {loading ? 'Signing in…' : 'Continue'}
          </button>
        </form>
        <p className="login-page__forgot">Forgot password?</p>
      </div>
    </div>
  );
}
