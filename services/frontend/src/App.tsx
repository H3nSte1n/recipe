import { useMemo, useState } from "react";
import { Home } from "./pages/Home";
import { Login } from "./pages/Login";
import { authApi } from "./services/api";

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(authApi.isAuthenticated());

  const content = useMemo(() => {
    if (!isAuthenticated) {
      return <Login onLoginSuccess={() => setIsAuthenticated(true)} />;
    }

    return <Home onLogout={() => setIsAuthenticated(false)} />;
  }, [isAuthenticated]);

  return content;
}

export default App

