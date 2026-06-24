import { useState } from 'react';
import { isAuthenticated as checkAuth } from './services/authService';
import LandingPage from './pages/LandingPage';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import HomePage from './pages/HomePage';

type Screen = 'landing' | 'login' | 'register' | 'home';

function App() {
  const [screen, setScreen] = useState<Screen>(checkAuth() ? 'home' : 'landing');

  if (screen === 'home') {
    return <HomePage onLogout={() => setScreen('landing')} />;
  }

  if (screen === 'login') {
    return (
      <LoginPage
        onLogin={() => setScreen('home')}
        onBack={() => setScreen('landing')}
      />
    );
  }

  if (screen === 'register') {
    return (
      <RegisterPage
        onRegister={() => setScreen('home')}
        onBack={() => setScreen('landing')}
      />
    );
  }

  return (
    <LandingPage
      onLogin={() => setScreen('login')}
      onRegister={() => setScreen('register')}
    />
  );
}

export default App;
