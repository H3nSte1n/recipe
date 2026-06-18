import { useAuth } from './hooks/useAuth';
import LoginPage from './pages/LoginPage';

function App() {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <LoginPage onLogin={() => {}} />;
  }

  return <div>Home — Phase 4</div>;
}

export default App;
