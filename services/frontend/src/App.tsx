import { useState } from 'react';
import { isAuthenticated as checkAuth } from './services/authService';
import LoginPage from './pages/LoginPage';
import HomePage from './pages/HomePage';

function App() {
  const [authed, setAuthed] = useState(checkAuth());

  if (!authed) {
    return <LoginPage onLogin={() => setAuthed(true)} />;
  }

  return <HomePage onLogout={() => setAuthed(false)} />;
}

export default App;
