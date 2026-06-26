"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api, clearTokens, loadTokens, setTokens } from "@/lib/api";
import type { TokenPair, User } from "@/lib/types";

type AuthContextType = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
  setAuthFromTokens: (tokens: TokenPair) => Promise<void>;
  resendVerification: (email: string) => Promise<void>;
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchMe = useCallback(async () => {
    try {
      const me = await api<User>("/auth/me");
      setUser(me);
    } catch {
      clearTokens();
      setUser(null);
    }
  }, []);

  useEffect(() => {
    loadTokens();
    fetchMe().finally(() => setLoading(false));
  }, [fetchMe]);

  const setAuthFromTokens = async (tokens: TokenPair) => {
    setTokens(tokens.access_token, tokens.refresh_token);
    await fetchMe();
  };

  const login = async (email: string, password: string) => {
    const data = await api<{ user: User; tokens: TokenPair }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
    setTokens(data.tokens.access_token, data.tokens.refresh_token);
    setUser(data.user);
  };

  const register = async (email: string, password: string, name: string) => {
    const data = await api<{ user: User; tokens: TokenPair }>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, name }),
    });
    setTokens(data.tokens.access_token, data.tokens.refresh_token);
    setUser(data.user);
  };

  const resendVerification = async (email: string) => {
    await api("/auth/resend-verification", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
  };

  const refreshUser = async () => {
    await fetchMe();
  };

  const logout = () => {
    clearTokens();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{ user, loading, login, register, logout, setAuthFromTokens, resendVerification, refreshUser }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
