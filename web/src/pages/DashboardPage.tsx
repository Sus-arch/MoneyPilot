import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { useAuth } from "../context/AuthContext";
import { get } from "../api/client";
import { CreditCard, Banknote, PiggyBank, Wallet, Landmark, Loader2 } from "lucide-react";

interface Account {
  account_id: string;
  nickname: string;
  currency: string;
  account_subtype: string;
  bank: string;
  balance?: number;
}

interface Recommendation {
  id: number;
  title: string;
  description: string;
  category: string;
  priority: "low" | "medium" | "high";
  created_at: string;
}

const ACCOUNT_SUBTYPE_RU: Record<string, string> = {
  Checking: "Текущий счёт",
  Savings: "Накопительный счёт",
  Loan: "Кредитный счёт",
  Card: "Карточный счёт",
  Deposit: "Вклад",
};

const ACCOUNT_ICONS: Record<string, React.ReactNode> = {
  Checking: <Landmark className="w-6 h-6 text-blue-600" />,
  Savings: <PiggyBank className="w-6 h-6 text-pink-500" />,
  Loan: <Banknote className="w-6 h-6 text-amber-600" />,
  Card: <CreditCard className="w-6 h-6 text-green-600" />,
  Deposit: <Wallet className="w-6 h-6 text-purple-600" />,
};

export default function DashboardPage() {
  const { currentBank, bankTokens } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [totalBalance, setTotalBalance] = useState<number>(0);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [errorAccounts, setErrorAccounts] = useState("");

  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loadingRecs, setLoadingRecs] = useState(false);
  const [errorRecs, setErrorRecs] = useState("");

  // 🔹 Получение счетов
  const fetchAccounts = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    setLoadingAccounts(true);
    setErrorAccounts("");

    try {
      const res = await get("/accounts", {
        Authorization: `Bearer ${bankTokens[currentBank]}`,
      });

      const rawAccounts: Account[] = res.accounts || [];

      const accountsWithBalances = await Promise.all(
        rawAccounts.map(async (acc) => {
          try {
            const balanceRes = await get(`/accounts/${acc.account_id}/balances`, {
              Authorization: `Bearer ${bankTokens[acc.bank || currentBank]}`,
              "X-Bank-Code": (acc.bank || currentBank).toLowerCase(),
            });

            const available = balanceRes.data?.balance?.find(
              (b: any) => b.type === "InterimAvailable"
            );
            const balance = available ? parseFloat(available.amount.amount) : 0;
            return { ...acc, balance };
          } catch {
            return { ...acc, balance: 0 };
          }
        })
      );

      setAccounts(accountsWithBalances);

      const total = accountsWithBalances
        .filter((a) => a.currency === "RUB")
        .reduce((sum, a) => sum + (a.balance || 0), 0);

      setTotalBalance(total);
    } catch (err) {
      console.error(err);
      setErrorAccounts("Не удалось загрузить счета");
    } finally {
      setLoadingAccounts(false);
    }
  };

  // 🔹 Получение рекомендаций
  const fetchRecommendations = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    setLoadingRecs(true);
    setErrorRecs("");

    try {
      const response = await fetch("http://localhost:8000/analyze", {
        headers: {
          Authorization: `Bearer ${bankTokens[currentBank]}`,
        },
      });

      if (!response.ok) throw new Error("Ошибка при загрузке рекомендаций");

      const data = await response.json();
      setRecommendations(data?.data || []);
    } catch (err) {
      console.error(err);
      setErrorRecs("Не удалось загрузить рекомендации");
    } finally {
      setLoadingRecs(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
    fetchRecommendations();
  }, [currentBank, bankTokens]);

  return (
    <motion.div
      className="max-w-6xl mx-auto p-8"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.6 }}
    >
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-8">
        Панель управления
      </h1>

      {/* Общий баланс */}
      <motion.div
        className="bg-blue-100 rounded-2xl p-6 mb-8 text-center shadow-lg"
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="text-xl font-semibold text-gray-700 mb-2">
          Общий баланс
        </h2>
        {loadingAccounts ? (
          <Loader2 className="w-6 h-6 mx-auto text-blue-700 animate-spin" />
        ) : (
          <p className="text-4xl font-bold text-blue-800">
            {totalBalance.toLocaleString("ru-RU", {
              style: "currency",
              currency: "RUB",
            })}
          </p>
        )}
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Счета */}
        <motion.div
          className="bg-white rounded-2xl p-6 shadow-md"
          initial={{ x: -50, opacity: 0 }}
          animate={{ x: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
        >
          <h2 className="text-2xl font-semibold text-gray-800 mb-4">
            Ваши счета
          </h2>

          {errorAccounts && (
            <p className="text-center text-red-500 mb-2">{errorAccounts}</p>
          )}

          {loadingAccounts ? (
            <div className="flex justify-center py-6">
              <Loader2 className="w-6 h-6 text-blue-600 animate-spin" />
            </div>
          ) : (
            <div className="space-y-3">
              {accounts.map((acc, i) => (
                <motion.div
                  key={`${acc.bank}-${acc.account_id}`}
                  className="p-4 border rounded-xl bg-gray-50 hover:shadow-md transition flex items-center justify-between"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3, delay: i * 0.1 }}
                >
                  <div className="flex items-center gap-3">
                    {ACCOUNT_ICONS[acc.account_subtype] || (
                      <Wallet className="w-6 h-6 text-gray-400" />
                    )}
                    <div>
                      <p className="font-semibold text-gray-800">
                        {ACCOUNT_SUBTYPE_RU[acc.account_subtype] ||
                          acc.account_subtype}
                      </p>
                    </div>
                  </div>
                  <p className="text-lg font-bold text-blue-700">
                    {acc.balance?.toLocaleString("ru-RU")} {acc.currency}
                  </p>
                </motion.div>
              ))}
            </div>
          )}
        </motion.div>

        {/* Рекомендации */}
        <motion.div
          className="bg-white rounded-2xl p-6 shadow-md"
          initial={{ x: 50, opacity: 0 }}
          animate={{ x: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          <h2 className="text-2xl font-semibold text-gray-800 mb-4">
            Персональные советы
          </h2>

          {errorRecs && (
            <p className="text-center text-red-500 mb-2">{errorRecs}</p>
          )}

          {loadingRecs ? (
            <div className="flex justify-center py-6">
              <Loader2 className="w-6 h-6 text-blue-600 animate-spin" />
            </div>
          ) : recommendations.length === 0 ? (
            <p className="text-center text-gray-600">
              Рекомендаций пока нет.
            </p>
          ) : (
            <div className="space-y-4">
              {recommendations.map((rec, i) => (
                <motion.div
                  key={rec.id}
                  className={`p-4 border rounded-xl ${
                    rec.priority === "high"
                      ? "border-red-400 bg-red-50"
                      : rec.priority === "medium"
                      ? "border-yellow-400 bg-yellow-50"
                      : "border-green-400 bg-green-50"
                  }`}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3, delay: 0.2 + i * 0.1 }}
                >
                  <h3 className="font-semibold text-lg text-gray-800 mb-1">
                    {rec.title}
                  </h3>
                  <p className="text-gray-600 text-sm mb-2">{rec.description}</p>
                  <p className="text-xs text-gray-500">
                    Категория: {rec.category} •{" "}
                    {new Date(rec.created_at).toLocaleDateString("ru-RU")}
                  </p>
                </motion.div>
              ))}
            </div>
          )}
        </motion.div>
      </div>
    </motion.div>
  );
}
