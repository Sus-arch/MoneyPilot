import { useState } from "react";
import { useAuth } from "../context/AuthContext";
import { post, del } from "../api/client";

interface Bank {
  id: string;
  name: string;
  connected: boolean;
}

const BANKS = [
  { id: "abank", name: "ABank" },
  { id: "vbank", name: "VBank" },
  { id: "sbank", name: "SBank" },
];

export default function BanksPage() {
  const { token } = useAuth();
  const [banks, setBanks] = useState<Bank[]>(
    BANKS.map((b) => ({
      ...b,
      connected: localStorage.getItem("connectedBank") === b.id,
    }))
  );
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  const connectBank = async (bankId: string) => {
    if (!token) return;
    setLoading(bankId);
    setMessage("");

    try {
      // Тело запроса пустое, банк передаётся только в заголовке
      await post(
        "/account-consent",
        undefined,
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": bankId,
        }
      );

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: true } : b))
      );

      setMessage(`✅ Банк ${bankId.toUpperCase()} успешно подключён`);
      localStorage.setItem("connectedBank", bankId);
    } catch (err) {
      setMessage("❌ Ошибка при подключении банка");
    } finally {
      setLoading(null);
    }
  };

  const disconnectBank = async (bankId: string) => {
    if (!token) return;
    setLoading(bankId);
    setMessage("");

    try {
      await del(
        "/account-consent",
        undefined,
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": bankId,
        }
      );

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: false } : b))
      );

      setMessage(`⚠️ Банк ${bankId.toUpperCase()} был отключён`);
      localStorage.removeItem("connectedBank");
    } catch (err) {
      setMessage("❌ Ошибка при отключении банка");
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
            {bank.connected ? (
              <button
                onClick={() => disconnectBank(bank.id)}
                disabled={!!loading}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition disabled:bg-gray-400"
              >
                {loading === bank.id ? "Отключение..." : "Отключить"}
              </button>
            ) : (
              <button
                onClick={() => connectBank(bank.id)}
                disabled={!!loading}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition disabled:bg-gray-400"
              >
                {loading === bank.id ? "Подключение..." : "Подключить"}
              </button>
            )}
          </div>
        ))}
      </div>

      {message && (
        <p className="text-center mt-6 text-gray-700 font-medium">{message}</p>
      )}
    </div>
  );
}
