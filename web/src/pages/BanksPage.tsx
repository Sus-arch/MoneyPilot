import { useState, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import { get, post, del } from "../api/client";

interface Bank {
  id: string;
  name: string;
  connected: boolean;
}

const BANKS = [
  { id: "vbank", name: "VBank" },
  { id: "abank", name: "ABank" },
  { id: "sbank", name: "SBank" },
];

export default function BanksPage() {
  const { currentBank, bankTokens, saveBankToken } = useAuth();
  const [banks, setBanks] = useState<Bank[]>(
    BANKS.map((b) => ({ ...b, connected: !!bankTokens[b.id] }))
  );
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [showLogin, setShowLogin] = useState<string | null>(null);
  const [credentials, setCredentials] = useState({ email: "", password: "" });

  // Функция для получения всех счетов текущим токеном
  const fetchAccounts = async () => {
    if (!currentBank) return;
    const token = bankTokens[currentBank];
    if (!token) return;

    try {
      const res = await get("/accounts", { Authorization: `Bearer ${token}` });

      // Обновляем connected для всех банков
      setBanks((prev) =>
        prev.map((b) => ({
          ...b,
          connected: res.accounts?.some((a: any) => a.bank === b.id) || false,
        }))
      );
    } catch {

    }
  };

  useEffect(() => {
    fetchAccounts();
  }, [currentBank, bankTokens]);

  const handleConnect = async (bankId: string) => {
    setShowLogin(bankId);
  };

  const handleLoginSubmit = async (bankId: string) => {
    setLoading(bankId);
    setMessage("");

    try {
      // логинимся и получаем JWT
      const response = await post("/auth/login", {
        email: credentials.email,
        password: credentials.password,
        bank: bankId,
      });

      const newJwt = response.token;
      if (!newJwt) throw new Error("JWT токен не получен");

      // сохраняем токен
      saveBankToken(bankId, newJwt);

      // создаём согласие
      const consentResponse = await post(
        "/account-consent",
        undefined,
        { "X-Bank-Code": bankId, Authorization: `Bearer ${newJwt}` }
      );

      if (consentResponse.auto_approved) {
  setMessage(`✅ Банк ${bankId.toUpperCase()} успешно подключён`);
  await fetchAccounts(); // обновляем accounts сразу
  setLoading(null);
} else {
  // ручное подтверждение
  setMessage(
    `⚠️ Для банка ${bankId.toUpperCase()} необходимо подтвердить согласие в приложении банка. Ожидание...`
  );

  const poll = setInterval(async () => {
  try {
    const res = await get("/accounts", { Authorization: `Bearer ${newJwt}` });

    const bankConnected = res.accounts?.some((a: any) => a.bank === bankId);
    if (bankConnected) {
      clearInterval(poll);
      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: true } : b))
      );
      setMessage(
        `✅ Согласие для ${bankId.toUpperCase()} подтверждено и банк подключён`
      );
      setLoading(null);
    }
  } catch {
    // ждём дальше
  }
}, 10000);

}

    } catch {
      setMessage("❌ Ошибка при подключении банка");
      setLoading(null);
    } finally {
      setShowLogin(null);
    }
  };

  const disconnectBank = async (bankId: string) => {
    const bankJwt = bankTokens[bankId];
    if (!bankJwt) return;

    setLoading(bankId);
    setMessage("");

    try {
      await del("/account-consent", undefined, {
        "X-Bank-Code": bankId,
        Authorization: `Bearer ${bankJwt}`,
      });

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: false } : b))
      );

      setMessage(`⚠️ Согласие для ${bankId.toUpperCase()} отозвано`);
      await fetchAccounts();
    } catch {
      setMessage("❌ Ошибка при отзыве согласия");
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="max-w-3xl mx-auto bg-white rounded-2xl shadow-md p-8 mt-10">
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-6">
        Управление банками
      </h1>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        {banks.map((bank) => (
          <div
            key={bank.id}
            className={`p-4 rounded-xl border flex flex-col items-center justify-between ${
              bank.connected
                ? "border-green-500 bg-green-50"
                : "border-gray-300 bg-gray-50"
            }`}
          >
            <h2 className="text-xl font-semibold mb-2">{bank.name}</h2>
            {bank.id === currentBank && (
              <p className="text-blue-600 font-semibold text-sm">(основной)</p>
            )}
            {bank.connected ? (
              <button
                onClick={() => disconnectBank(bank.id)}
                disabled={!!loading}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition disabled:bg-gray-400 mt-2"
              >
                {loading === bank.id ? "Отключение..." : "Отключить"}
              </button>
            ) : (
              <button
                onClick={() => handleConnect(bank.id)}
                disabled={!!loading}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition disabled:bg-gray-400 mt-2"
              >
                {loading === bank.id ? "Подключение..." : "Подключить"}
              </button>
            )}
          </div>
        ))}
      </div>

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
