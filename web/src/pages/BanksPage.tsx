import { useState, useRef } from "react";
import { useAuth } from "../context/AuthContext";
import { post, del } from "../api/client";
import { Loader2, Banknote, CreditCard, Wallet } from "lucide-react";

interface Bank {
  id: string;
  name: string;
  connected: boolean;
  status?: "idle" | "pending";
}

const BANKS = [
  { id: "vbank", name: "VBank", icon: <Banknote className="w-8 h-8 text-blue-600" /> },
  { id: "abank", name: "ABank", icon: <CreditCard className="w-8 h-8 text-green-600" /> },
  { id: "sbank", name: "SBank", icon: <Wallet className="w-8 h-8 text-purple-600" /> },
];

export default function BanksPage() {
  const { currentBank, bankTokens, saveBankToken } = useAuth();
  const [banks, setBanks] = useState<Bank[]>(
    BANKS.map((b) => ({ ...b, connected: !!bankTokens[b.id], status: "idle" }))
  );
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [showLogin, setShowLogin] = useState<string | null>(null);
  const [credentials, setCredentials] = useState({ email: "", password: "" });
  const wsRef = useRef<WebSocket | null>(null);
  const activeBankRef = useRef<string | null>(null); // <- сохраняем, к какому банку относится сокет

  // 🔹 Подключение банка (открывает окно логина)
  const handleConnect = (bankId: string) => setShowLogin(bankId);

  // 🔹 Отправка логина
  const handleLoginSubmit = async (bankId: string) => {
    setLoading(bankId);
    setMessage("");

    try {
      // 1️⃣ Авторизация
      const response = await post("/auth/login", {
        email: credentials.email,
        password: credentials.password,
        bank: bankId,
      });

      const token = response.token;
      if (!token) throw new Error("JWT токен не получен");

      saveBankToken(bankId, token);

      // 2️⃣ Создание согласия
      const consentResponse = await post(
        "/account-consent",
        undefined,
        { "X-Bank-Code": bankId, Authorization: `Bearer ${token}` }
      );

      // если согласие approved — сразу подключаем
      if (consentResponse.status === "approved") {
        setBanks((prev) =>
          prev.map((b) => (b.id === bankId ? { ...b, connected: true, status: "idle" } : b))
        );
        setMessage(`✅ Банк ${bankId.toUpperCase()} успешно подключён`);
        setLoading(null);
        return;
      }

      // иначе pending — ждём подтверждения через WebSocket
      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, status: "pending" } : b))
      );
      setMessage(`⚠️ Требуется подтверждение в приложении ${bankId.toUpperCase()}`);

      // 3️⃣ Подключаем WebSocket для ожидания события
      connectWebSocket(bankId);
    } catch (err) {
      console.error(err);
      setMessage("❌ Ошибка при подключении банка");
    } finally {
      setLoading(null);
      setShowLogin(null);
    }
  };

  // 🔹 Отключение банка
  const disconnectBank = async (bankId: string) => {
    const token = bankTokens[bankId];
    if (!token) return;

    setLoading(bankId);
    setMessage("");

    try {
      await del("/account-consent", undefined, {
        "X-Bank-Code": bankId,
        Authorization: `Bearer ${token}`,
      });

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: false, status: "idle" } : b))
      );

      setMessage(`⚠️ Согласие для ${bankId.toUpperCase()} отозвано`);
    } catch (err) {
      console.error(err);
      setMessage("❌ Ошибка при отзыве согласия");
    } finally {
      setLoading(null);
    }
  };

  // 🔹 Подключение WebSocket (только при подключении банка)
  const connectWebSocket = (bankId: string) => {
    if (wsRef.current) wsRef.current.close();

    const socket = new WebSocket("ws://localhost:8080/ws");
    wsRef.current = socket;
    activeBankRef.current = bankId;

    socket.onopen = () => console.log(`✅ WebSocket для ${bankId} открыт`);
    socket.onclose = () => {
      console.log(`❌ WebSocket закрыт для ${bankId}`);
      wsRef.current = null;
      activeBankRef.current = null;
    };

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        console.log("📨 WebSocket message:", msg);

        // В ответе только consent_id и status
        if (!msg.consent_id || !msg.status) return;

        const currentBank = activeBankRef.current;
        if (!currentBank) return;

        if (msg.status === "pending") {
          setBanks((prev) =>
            prev.map((b) => (b.id === currentBank ? { ...b, status: "pending" } : b))
          );
          setMessage(`⚠️ Подтверждение для ${currentBank.toUpperCase()} ожидается`);
        }

        if (msg.status === "approved") {
          setBanks((prev) =>
            prev.map((b) =>
              b.id === currentBank ? { ...b, connected: true, status: "idle" } : b
            )
          );
          setMessage(`✅ Согласие для ${currentBank.toUpperCase()} подтверждено`);
          socket.close();
        }
      } catch (err) {
        console.error("Ошибка при обработке WebSocket-сообщения:", err);
      }
    };
  };

  // 🔹 UI
  return (
    <div className="max-w-4xl mx-auto mt-10 p-6">
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-8">
        Управление банками
      </h1>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        {banks.map((bank) => {
          const Icon = BANKS.find((b) => b.id === bank.id)?.icon;
          const isPending = bank.status === "pending";
          return (
            <div
              key={bank.id}
              className={`p-6 rounded-2xl border shadow-md flex flex-col items-center justify-between transition transform hover:-translate-y-1 hover:shadow-lg relative ${
                bank.connected
                  ? "border-green-500 bg-green-50"
                  : isPending
                  ? "border-yellow-400 bg-yellow-50 animate-pulse"
                  : "border-gray-300 bg-gray-50"
              }`}
            >
              {isPending && (
                <div className="absolute top-4 right-4 animate-spin">
                  <Loader2 className="w-5 h-5 text-yellow-500" />
                </div>
              )}
              {Icon}
              <h2 className="text-xl font-semibold mt-3">{bank.name}</h2>

              {bank.id === currentBank && (
                <span className="text-blue-600 font-medium text-sm mt-1">
                  (основной)
                </span>
              )}

              {bank.connected ? (
                <button
                  onClick={() => disconnectBank(bank.id)}
                  disabled={!!loading || isPending}
                  className="mt-4 px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition disabled:bg-gray-400"
                >
                  {loading === bank.id ? "Отключение..." : "Отключить"}
                </button>
              ) : (
                <button
                  onClick={() => handleConnect(bank.id)}
                  disabled={!!loading || isPending}
                  className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:bg-gray-400"
                >
                  {loading === bank.id ? "Подключение..." : "Подключить"}
                </button>
              )}
            </div>
          );
        })}
      </div>

      {/* 🔹 Модальное окно логина */}
      {showLogin && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-gray-800 p-6 rounded-xl shadow-lg w-96 text-white">
            <h2 className="text-xl font-bold mb-4">
              Вход в {showLogin.toUpperCase()}
            </h2>

            <input
              type="text"
              placeholder="Логин"
              value={credentials.email}
              onChange={(e) =>
                setCredentials({ ...credentials, email: e.target.value })
              }
              className="border border-gray-600 w-full mb-3 px-3 py-2 rounded bg-gray-700 text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            <input
              type="password"
              placeholder="Пароль"
              value={credentials.password}
              onChange={(e) =>
                setCredentials({ ...credentials, password: e.target.value })
              }
              className="border border-gray-600 w-full mb-3 px-3 py-2 rounded bg-gray-700 text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowLogin(null)}
                className="px-3 py-1 rounded bg-gray-500 hover:bg-gray-600 transition"
              >
                Отмена
              </button>
              <button
                onClick={() => handleLoginSubmit(showLogin)}
                className="px-3 py-1 rounded bg-blue-600 text-white hover:bg-blue-700 transition"
              >
                Подключить
              </button>
            </div>
          </div>
        </div>
      )}

      {message && (
        <p className="text-center mt-6 text-gray-700 font-medium">{message}</p>
      )}
    </div>
  );
}
