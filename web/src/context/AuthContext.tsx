import { createContext, useState, useContext } from "react";
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { post } from "../api/client";

interface AuthContextType {
  token: string | null;
  currentBank: string | null;
  bankTokens: Record<string, string>;
  login: (email: string, password: string, bank: string) => Promise<"ok" | "waiting">;
  logout: () => void;
  saveBankToken: (bank: string, token: string) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(localStorage.getItem("token"));
  const [currentBank, setCurrentBank] = useState<string | null>(
    localStorage.getItem("currentBank")
  );
  const [bankTokens, setBankTokens] = useState<Record<string, string>>(
    JSON.parse(localStorage.getItem("bankTokens") || "{}")
  );
  const navigate = useNavigate();

  const saveBankToken = (bank: string, jwt: string) => {
    const updated = { ...bankTokens, [bank]: jwt };
    setBankTokens(updated);
    localStorage.setItem("bankTokens", JSON.stringify(updated));
  };

  const login = async (email: string, password: string, bank: string): Promise<"ok" | "waiting"> => {
    // 🔹 1. Логинимся
    const response = await post("/auth/login", { email, password, bank });
    const jwt = response.token;
    if (!jwt) throw new Error("JWT токен не получен");

    // 🔹 2. Сохраняем токен
    setToken(jwt);
    setCurrentBank(bank);
    localStorage.setItem("token", jwt);
    localStorage.setItem("currentBank", bank);

    // 🔹 3. Сохраняем JWT для этого банка
    saveBankToken(bank, jwt);

    // 🔹 4. Создаём согласие
    const consentResponse = await post("/account-consent", undefined, {
      "X-Bank-Code": bank,
      Authorization: `Bearer ${jwt}`,
    });

    // 🔹 5. Если банк — sbank и согласие ручное
    if (bank === "sbank" && !consentResponse.auto_approved) {
      console.log("Ожидаем ручного подтверждения согласия от SBank...");

      // ⚠️ возвращаем статус ожидания, чтобы фронт показал сообщение
      return "waiting";
    }

    // 🔹 6. Для остальных — сразу переход
    navigate("/dashboard");
    return "ok";
  };

  const logout = () => {
    setToken(null);
    setCurrentBank(null);
    setBankTokens({});
    localStorage.removeItem("token");
    localStorage.removeItem("currentBank");
    localStorage.removeItem("bankTokens");
    navigate("/login");
  };

  return (
    <AuthContext.Provider
      value={{ token, currentBank, bankTokens, login, logout, saveBankToken }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within an AuthProvider");
  return context;
};
