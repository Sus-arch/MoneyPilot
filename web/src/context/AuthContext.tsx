import { createContext, useState, useContext } from "react";
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { post } from "../api/client";

interface AuthContextType {
  token: string | null;
  login: (email: string, password: string, bank: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(
    localStorage.getItem("token")
  );
  const navigate = useNavigate();

  const login = async (email: string, password: string, bank: string) => {
    try {
      const response = await post("/auth/login", { email, password, bank });
      const { token } = response;

      localStorage.setItem("token", token);
      setToken(token);

      // Редирект после успешного логина
      navigate("/dashboard");
    } catch (err) {
      throw new Error("Ошибка авторизации");
    }
  };

  const logout = () => {
    localStorage.removeItem("token");
    setToken(null);
    navigate("/login");
  };

  return (
    <AuthContext.Provider value={{ token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};