import { createContext, useState, useContext, useRef } from "react";
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

  // 🔹 WebSocket для ожидания согласия
  const wsRef = useRef<WebSocket | null>(null);
  const activeBankRef = useRef<string | null>(null);

  const saveBankToken = (bank: string, jwt: string) => {
    const updated = { ...bankTokens, [bank]: jwt };
    setBankTokens(updated);
    localStorage.setItem("bankTokens", JSON.stringify(updated));
  };

  const login = async (email: string, password: string, bank: string): Promise<"ok" | "waiting"> => {
    // 1️⃣ Авторизация
    const response = await post("/auth/login", { email, password, bank });
    const jwt = response.token;
    if (!jwt) throw new Error("JWT токен не получен");

    setToken(jwt);
    setCurrentBank(bank);
    localStorage.setItem("token", jwt);
    localStorage.setItem("currentBank", bank);
    saveBankToken(bank, jwt);

    // 2️⃣ Создание согласия
    const consentResponse = await post("/account-consent", undefined, {
      "X-Bank-Code": bank,
      Authorization: `Bearer ${jwt}`,
    });

    // 3️⃣ Если согласие auto_approved — сразу переход
    if (consentResponse.status === "approved" || consentResponse.auto_approved) {
      navigate("/dashboard");
      return "ok";
    }

    // 4️⃣ Иначе — подключаем WebSocket для ожидания ручного согласия
    if (wsRef.current) wsRef.current.close();
    const socket = new WebSocket("ws://localhost:8080/ws");
    wsRef.current = socket;
    activeBankRef.current = bank;

    socket.onopen = () => console.log(`✅ WebSocket для ${bank} открыт`);
    socket.onclose = () => {
      console.log(`❌ WebSocket закрыт для ${bank}`);
      wsRef.current = null;
      activeBankRef.current = null;
    };

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (!msg.consent_id || !msg.status) return;

        const current = activeBankRef.current;
        if (!current) return;

        if (msg.status === "approved") {
          saveBankToken(current, jwt);
          setToken(jwt);
          setCurrentBank(current);
          navigate("/dashboard");
          socket.close();
        }
      } catch (err) {
        console.error("Ошибка при обработке WebSocket-сообщения:", err);
      }
    };

    return "waiting";
  };

  const logout = () => {
    setToken(null);
    setCurrentBank(null);
    setBankTokens({});
    localStorage.clear();
    navigate("/login");
    if (wsRef.current) wsRef.current.close();
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
