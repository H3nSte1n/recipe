import ScatteredBackground from '../components/ScatteredBackground';
import '../styles/LandingPage.css';

interface LandingPageProps {
  onLogin: () => void;
  onRegister: () => void;
}

export default function LandingPage({ onLogin, onRegister }: LandingPageProps) {
  return (
    <div className="landing-page">
      <ScatteredBackground />
      <div className="landing-page__center">
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
      </div>
    </div>
  );
}
