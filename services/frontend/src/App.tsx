import { useState } from 'react';
import { isAuthenticated as checkAuth } from './services/authService';
import LandingPage from './pages/LandingPage';
import HomePage from './pages/HomePage';

type Screen = 'landing' | 'home';

function App() {
  const [screen, setScreen] = useState<Screen>(checkAuth() ? 'home' : 'landing');

  if (screen === 'home') {
    return <HomePage onLogout={() => setScreen('landing')} />;
  }

  return (
    <LandingPage
      onLogin={() => setScreen('home')}
      onRegister={() => setScreen('home')}
    />
  );
}

export default App;
