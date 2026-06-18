import { useState } from 'react';
import * as authService from '../services/authService';

interface UseAuthReturn {
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

export function useAuth(): UseAuthReturn {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(
    authService.isAuthenticated(),
  );

  const login = async (email: string, password: string): Promise<void> => {
    await authService.login(email, password);
    setIsAuthenticated(true);
  };

  const logout = (): void => {
    authService.logout();
    setIsAuthenticated(false);
  };

  return { isAuthenticated, login, logout };
}
