import { useState } from 'react';
import { register } from '../services/authService';
import ScatteredBackground from '../components/ScatteredBackground';
import '../styles/RegisterPage.css';

interface RegisterPageProps {
  onRegister: () => void;
  onBack: () => void;
}

export default function RegisterPage({ onRegister, onBack }: RegisterPageProps) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await register(name, email, password);
      onRegister();
    } catch {
      setError('Registration failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="register-page">
      <ScatteredBackground />
      <div className="register-page__form-area">
        <button className="register-page__back" onClick={onBack} type="button" aria-label="Back">
          ←
        </button>
        <p className="register-page__label">Create account</p>
        <form className="register-page__form" onSubmit={handleSubmit}>
          <input
            className="register-page__input"
            type="text"
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            autoComplete="name"
          />
          <input
            className="register-page__input"
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
          <input
            className="register-page__input"
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
          />
          {error && <p className="register-page__error">{error}</p>}
          <button className="register-page__submit" type="submit" disabled={loading}>
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </form>
        <p className="register-page__forgot">Already have an account? <span onClick={onBack} style={{ cursor: 'pointer' }}>Sign in</span></p>
      </div>
    </div>
  );
}
