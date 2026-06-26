import { useEffect, useRef } from 'react';
import ScatteredBackground from '../components/ScatteredBackground';
import '../styles/LandingPage.css';

interface LandingPageProps {
  onLogin: () => void;
  onRegister: () => void;
}

export default function LandingPage({ onLogin, onRegister }: LandingPageProps) {
  const centerRef = useRef<HTMLDivElement>(null);

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
      <ScatteredBackground />
      <div ref={centerRef} className="landing-page__center">
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
