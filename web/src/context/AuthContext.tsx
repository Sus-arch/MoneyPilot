import { createContext, useState, useContext, useEffect } from "react";
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { post, get } from "../api/client";
import jwtDecode from "jwt-decode";

interface User {
  id: string;
  email: string;
  bank: string;
}

interface AuthContextType {
  token: string | null;
  user: User | null;
  login: (email: string, password: string, bank: string) => Promise<void>;
  logout: () => void;
  loadUser: (token?: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(
    localStorage.getItem("token")
  );
  const [user, setUser] = useState<User | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (token) loadUser(token);
  }, [token]);

  const login = async (email: string, password: string, bank: string) => {
    try {
      const response = await post("/auth/login", { email, password, bank });
      const { token } = response;

      localStorage.setItem("token", token);
      setToken(token);

      await loadUser(token);

      navigate("/dashboard"); // редирект после успешного логина
    } catch (err) {
      throw new Error("Ошибка авторизации");
    }
  };

  const logout = () => {
    localStorage.removeItem("token");
    setToken(null);
    setUser(null);
    navigate("/login");
  };

  const loadUser = async (tokenParam?: string) => {
    const t = tokenParam || token;
    if (!t) return;

    try {
      // Декодируем JWT
      const decoded: any = jwtDecode(t);

      // Получаем данные пользователя с сервера, передавая токен в заголовке Authorization
      const userData = await get(`/users/${decoded.id}`, {
        Authorization: `Bearer ${t}`,
      });

      setUser(userData);
    } catch (error) {
      console.error("Ошибка при загрузке пользователя:", error);
      logout();
    }
  };

  return (
    <AuthContext.Provider value={{ token, user, login, logout, loadUser }}>
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
