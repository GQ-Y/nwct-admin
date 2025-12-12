import React, { createContext, useContext, useMemo, useState } from "react";
import { api, clearToken, getToken, setToken } from "../lib/api";

type AuthState = {
  token: string | null;
  initialized: boolean | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  refreshInitStatus: () => Promise<boolean>;
};

const Ctx = createContext<AuthState | null>(null);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setTokenState] = useState<string | null>(getToken());
  const [initialized, setInitialized] = useState<boolean | null>(null);

  const refreshInitStatus = async () => {
    const st = await api.initStatus();
    setInitialized(st.initialized);
    return st.initialized;
  };

  const login = async (username: string, password: string) => {
    const res = await api.login(username, password);
    setToken(res.token);
    setTokenState(res.token);
    await refreshInitStatus();
  };

  const logout = () => {
    clearToken();
    setTokenState(null);
    setInitialized(null);
  };

  const value = useMemo<AuthState>(
    () => ({ token, initialized, login, logout, refreshInitStatus }),
    [token, initialized]
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
};

export function useAuth() {
  const v = useContext(Ctx);
  if (!v) throw new Error("AuthProvider missing");
  return v;
}


