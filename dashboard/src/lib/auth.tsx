"use client";

import React, { createContext, useContext, useEffect, useState, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";

interface AuthState {
  apiUrl: string;
  token: string;
  isAuthenticated: boolean;
  isLoading: boolean;
  connected: boolean;
}

interface AuthContextType extends AuthState {
  login: (apiUrl: string, token: string) => void;
  logout: () => void;
  setConnected: (v: boolean) => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

const STORAGE_KEY_URL = "qsim_api_url";
const STORAGE_KEY_TOKEN = "qsim_token";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [state, setState] = useState<AuthState>({
    apiUrl: "",
    token: "",
    isAuthenticated: false,
    isLoading: true,
    connected: false,
  });

  useEffect(() => {
    const url = localStorage.getItem(STORAGE_KEY_URL);
    const tok = localStorage.getItem(STORAGE_KEY_TOKEN);
    if (url && tok) {
      setState({ apiUrl: url, token: tok, isAuthenticated: true, isLoading: false, connected: false });
      // Check connectivity
      fetch(`${url}/health`, { headers: { Authorization: `Bearer ${tok}` } })
        .then((r) => {
          if (r.ok) setState((s) => ({ ...s, connected: true }));
        })
        .catch(() => {});
    } else {
      setState((s) => ({ ...s, isLoading: false }));
    }
  }, []);

  useEffect(() => {
    if (!state.isLoading && !state.isAuthenticated && pathname !== "/login") {
      router.push("/login");
    }
  }, [state.isLoading, state.isAuthenticated, pathname, router]);

  const login = useCallback((apiUrl: string, token: string) => {
    localStorage.setItem(STORAGE_KEY_URL, apiUrl);
    localStorage.setItem(STORAGE_KEY_TOKEN, token);
    setState({ apiUrl, token, isAuthenticated: true, isLoading: false, connected: true });
    router.push("/");
  }, [router]);

  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY_URL);
    localStorage.removeItem(STORAGE_KEY_TOKEN);
    setState({ apiUrl: "", token: "", isAuthenticated: false, isLoading: false, connected: false });
    router.push("/login");
  }, [router]);

  const setConnected = useCallback((v: boolean) => {
    setState((s) => ({ ...s, connected: v }));
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, logout, setConnected }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
