import { useState } from 'react';
import { isAuthenticated as checkAuth, logout as clearSession } from './services/authService';
import LandingPage from './pages/LandingPage';
import HomePage from './pages/HomePage';
import ProfilePage from './pages/ProfilePage';

type Screen = 'landing' | 'home' | 'profile';

function App() {
  const [screen, setScreen] = useState<Screen>(checkAuth() ? 'home' : 'landing');

  const handleLogout = (): void => {
    // Clear the JWT from localStorage before switching screens, otherwise a page
    // refresh would silently re-authenticate from the still-stored token.
    clearSession();
    setScreen('landing');
  };

  if (screen === 'home') {
    return <HomePage onLogout={handleLogout} onProfile={() => setScreen('profile')} />;
  }

  if (screen === 'profile') {
    return <ProfilePage onBack={() => setScreen('home')} onLogout={handleLogout} />;
  }

  return (
    <LandingPage
      onLogin={() => setScreen('home')}
      onRegister={() => setScreen('home')}
    />
  );
}

export default App;
