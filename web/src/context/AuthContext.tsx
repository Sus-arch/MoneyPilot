import { createContext, useState, useContext, useEffect } from "react";
import type { ReactNode } from "react";
import { post, get } from "../api/client";
import { useNavigate } from "react-router-dom";
import jwt_decode from "jwt-decode";

// Тип пользователя
export interface User {
  id: string;
  name?: string;
  [key: string]: any;
}

// Тип контекста
interface AuthContextType {
  token: string;
  user: User | null;
  loading: boolean;
  login: (id: string, password: string) => Promise<void>;
  logout: () => void;
  loadUser: () => Promise<void>;
}

// Контекст
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Провайдер
interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const navigate = useNavigate();
  const [token, setToken] = useState<string>(localStorage.getItem("token") || "");
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (token) {
      try {
        const decoded: any = jwt_decode(token); // Декодируем JWT
        const now = Date.now() / 1000;
        if (decoded.exp && decoded.exp < now) {
          logout();
        } else {
          loadUser();
        }
      } catch {
        logout();
      }
    } else {
      setLoading(false);
    }
  }, []);

  // Вход
  const login = async (id: string, password: string) => {
    const res = await post<{ token: string }>("/login", { id, password });
    if (!res.token) throw new Error("No token from backend");

    setToken(res.token);
    localStorage.setItem("token", res.token);
    await loadUser();
    navigate("/dashboard");
  };

  // Выход
  const logout = () => {
    setToken("");
    setUser(null);
    localStorage.removeItem("token");
    navigate("/login");
  };

  // Загрузка профиля
  const loadUser = async () => {
    try {
      const profile: User = await get("/me");
      setUser(profile);
    } catch (err) {
      console.error(err);
      logout();
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider value={{ token, user, loading, login, logout, loadUser }}>
      {children}
    </AuthContext.Provider>
  );
};

// Хук для использования контекста
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within AuthProvider");
  return context;
};
