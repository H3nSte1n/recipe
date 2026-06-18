import { useState } from 'react';
import { isAuthenticated as checkAuth } from './services/authService';
import LoginPage from './pages/LoginPage';

function App() {
  const [authed, setAuthed] = useState(checkAuth());

  if (!authed) {
    return <LoginPage onLogin={() => setAuthed(true)} />;
  }

  return <div>Home — Phase 4</div>;
}

export default App;
