import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { User, AuthTokens, Persona } from '../types';
import { api, setAccessToken, setRefreshToken, getRefreshToken } from '../api/client';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (payload: {
    email: string;
    password: string;
    full_name: string;
    persona?: Persona;
    timezone?: string;
    work_start?: string;
    work_end?: string;
  }) => Promise<void>;
  logout: () => void;
  updateProfile: (data: Partial<User>) => Promise<User>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchProfile = useCallback(async (): Promise<User> => {
    const res = await api.get<User>('/auth/me');
    setUser(res.data);
    return res.data;
  }, []);

  const logout = useCallback(() => {
    setAccessToken(null);
    setRefreshToken(null);
    setUser(null);
  }, []);

  // Check initial session with stored refresh token
  useEffect(() => {
    const initAuth = async () => {
      const storedRefreshToken = getRefreshToken();
      if (!storedRefreshToken) {
        setIsLoading(false);
        return;
      }

      try {
        const refreshRes = await fetch('/api/v1/auth/refresh', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: storedRefreshToken }),
        });

        if (!refreshRes.ok) {
          throw new Error('Initial session refresh failed');
        }

        const data: AuthTokens = await refreshRes.json();
        setAccessToken(data.access_token);
        setRefreshToken(data.refresh_token);
        await fetchProfile();
      } catch {
        logout();
      } finally {
        setIsLoading(false);
      }
    };

    initAuth();

    const handleExpired = () => logout();
    window.addEventListener('auth:expired', handleExpired);
    return () => window.removeEventListener('auth:expired', handleExpired);
  }, [fetchProfile, logout]);

  const login = async (email: string, password: string) => {
    const res = await api.post<AuthTokens>('/auth/login', { email, password });
    setAccessToken(res.data.access_token);
    setRefreshToken(res.data.refresh_token);
    await fetchProfile();
  };

  const register = async (payload: {
    email: string;
    password: string;
    full_name: string;
    persona?: Persona;
    timezone?: string;
    work_start?: string;
    work_end?: string;
  }) => {
    // 1. Register user
    await api.post<User>('/auth/register', payload);
    // 2. Automatically log in to get tokens
    await login(payload.email, payload.password);
  };

  const updateProfile = async (data: Partial<User>): Promise<User> => {
    const res = await api.put<User>('/auth/me', data);
    setUser(res.data);
    return res.data;
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
        updateProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
