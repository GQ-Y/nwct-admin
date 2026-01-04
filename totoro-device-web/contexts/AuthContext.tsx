import React, { createContext, useContext, useMemo, useState, useEffect } from "react";
import { api, clearToken, getToken, setToken, setTokenExpiredCallback } from "../lib/api";

type AuthState = {
  token: string | null;
  initialized: boolean | null;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  refreshInitStatus: () => Promise<boolean>;
};

const Ctx = createContext<AuthState | null>(null);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setTokenState] = useState<string | null>(getToken());
  const [initialized, setInitialized] = useState<boolean | null>(null);

  // 监听 token 清除事件
  useEffect(() => {
    const handleTokenCleared = () => {
      setTokenState(null);
      setInitialized(null);
    };

    window.addEventListener("token-cleared", handleTokenCleared);
    return () => {
      window.removeEventListener("token-cleared", handleTokenCleared);
    };
  }, []);

  // 设置 token 失效回调
  useEffect(() => {
    setTokenExpiredCallback(() => {
      setTokenState(null);
      setInitialized(null);
    });
  }, []);

  const refreshInitStatus = async () => {
    const st = await api.initStatus();
    setInitialized(st.initialized);
    return st.initialized;
  };

  const login = async (username: string, password: string) => {
    const res = await api.login(username, password);
    setToken(res.token);
    setTokenState(res.token);
    return await refreshInitStatus();
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


