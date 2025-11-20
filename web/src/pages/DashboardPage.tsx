import React, { useEffect, useState, useRef } from "react";
import { motion } from "framer-motion";
import { useAuth } from "../context/AuthContext";
import { get, clearCache } from "../api/client";
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

interface Affordability {
  can_afford: boolean;
  level: "SUCCESS" | "WARNING" | "CAUTION" | "CRITICAL";
  message: string;
  details: string;
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

  const [purchaseAmount, setPurchaseAmount] = useState<string>("");
  const [affordability, setAffordability] = useState<Affordability | null>(null);
  const [loadingAfford, setLoadingAfford] = useState(false);
  const [errorAfford, setErrorAfford] = useState("");

  const [spendingForecast, setSpendingForecast] = useState<{
    forecast: number;
    details: {
      variable_spending: number;
      fixed_obligations: number;
    };
    next_month: string;
  } | null>(null);
  const [loadingForecast, setLoadingForecast] = useState(false);
  const [errorForecast, setErrorForecast] = useState("");

  // Состояние подписки, сохраняем в localStorage
  const [isSubscribed, setIsSubscribed] = useState<boolean>(() => {
    return localStorage.getItem("isSubscribed") === "true";
  });

  const handleSubscribe = () => {
    setIsSubscribed(true);
    localStorage.setItem("isSubscribed", "true");
  };

  const fetchingAccountsRef = useRef(false);
  const lastBankRef = useRef<string | null>(null);

  // 🔹 Получение счетов
  const fetchAccounts = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    
    // Предотвращаем дублирующие запросы
    if (fetchingAccountsRef.current) return;
    if (lastBankRef.current === currentBank && accounts.length > 0) return;
    
    fetchingAccountsRef.current = true;
    setLoadingAccounts(true);
    setErrorAccounts("");

    try {
      // Получаем список всех подключенных банков
      const connectedBanks = Object.keys(bankTokens).filter(bank => bankTokens[bank]);
      const bankCodesHeader = connectedBanks.join(",");

      const res = await get("/accounts", {
        Authorization: `Bearer ${bankTokens[currentBank]}`,
        "X-Bank-Code": bankCodesHeader,
      });

      const rawAccounts: Account[] = res.accounts || [];

      // Ограничиваем параллельные запросы балансов (максимум 3 одновременно)
      const accountsWithBalances: Account[] = [];
      const batchSize = 3;
      
      for (let i = 0; i < rawAccounts.length; i += batchSize) {
        const batch = rawAccounts.slice(i, i + batchSize);
        const batchResults = await Promise.all(
          batch.map(async (acc) => {
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
            } catch (err: any) {
              // Игнорируем ошибки для отдельных балансов
              return { ...acc, balance: 0 };
            }
          })
        );
        accountsWithBalances.push(...batchResults);
      }

      setAccounts(accountsWithBalances);
      lastBankRef.current = currentBank;

      const total = accountsWithBalances
        .filter((a) => a.currency === "RUB")
        .reduce((sum, a) => sum + (a.balance || 0), 0);

      setTotalBalance(total);
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setErrorAccounts("Слишком много запросов. Подождите немного.");
      } else {
        setErrorAccounts("Не удалось загрузить счета");
      }
    } finally {
      setLoadingAccounts(false);
      fetchingAccountsRef.current = false;
    }
  };

  const fetchingRecsRef = useRef(false);

  // 🔹 Получение рекомендаций
  const fetchRecommendations = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    
    // Предотвращаем дублирующие запросы
    if (fetchingRecsRef.current) return;
    
    fetchingRecsRef.current = true;
    setLoadingRecs(true);
    setErrorRecs("");

    try {
      const response = await fetch("http://localhost:8000/analyze", {
        headers: {
          Authorization: `Bearer ${bankTokens[currentBank]}`,
        },
      });

      if (!response.ok) {
        if (response.status === 429) {
          throw new Error("Rate limit exceeded");
        }
        throw new Error("Ошибка при загрузке рекомендаций");
      }

      const data = await response.json();
      setRecommendations(data?.data || []);
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setErrorRecs("Слишком много запросов. Подождите немного.");
      } else {
        setErrorRecs("Не удалось загрузить рекомендации");
      }
    } finally {
      setLoadingRecs(false);
      fetchingRecsRef.current = false;
    }
  };

  // 🔹 Получение прогноза расходов
  const fetchSpendingForecast = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    
    setLoadingForecast(true);
    setErrorForecast("");
    setSpendingForecast(null);

    try {
      const response = await fetch("http://localhost:8000/predict_spending", {
        headers: {
          Authorization: `Bearer ${bankTokens[currentBank]}`,
        },
      });

      if (!response.ok) throw new Error("Ошибка при получении прогноза");

      const data = await response.json();
      if (data.status === "success") {
        setSpendingForecast({
          forecast: data.forecast,
          details: data.details,
          next_month: data.next_month,
        });
      } else {
        throw new Error(data.message || "Ошибка при получении прогноза");
      }
    } catch (err) {
      console.error(err);
      setErrorForecast("Не удалось получить прогноз расходов");
    } finally {
      setLoadingForecast(false);
    }
  };

  // 🔹 Проверка возможности покупки
  const checkAffordability = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    setLoadingAfford(true);
    setErrorAfford("");
    setAffordability(null);

    try {
      const response = await fetch(
        `http://localhost:8000/can_afford?amount=${purchaseAmount}`,
        {
          headers: {
            Authorization: `Bearer ${bankTokens[currentBank]}`,
          },
        }
      );

      if (!response.ok) throw new Error("Ошибка при проверке покупки");

      const data = await response.json();
      if (data.status === "success" && data.data) {
        setAffordability(data.data);
      } else {
        throw new Error(data.message || "Ошибка при проверке покупки");
      }
    } catch (err) {
      console.error(err);
      setErrorAfford("Не удалось проверить покупку");
    } finally {
      setLoadingAfford(false);
    }
  };

  useEffect(() => {
    // Очищаем кэш при смене банка
    if (lastBankRef.current && lastBankRef.current !== currentBank) {
      clearCache("/accounts");
      lastBankRef.current = null;
    }
    fetchAccounts();
    fetchRecommendations();
    fetchSpendingForecast();
  }, [currentBank]);

  // Ограничение рекомендаций для обычных пользователей
  const displayedRecommendations = isSubscribed ? recommendations : recommendations.slice(0, 2);

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

      {/* Общий баланс и прогноз расходов */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <motion.div
          className="bg-blue-100 rounded-2xl p-6 text-center shadow-lg"
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

        <motion.div
          className="bg-gradient-to-br from-purple-100 to-pink-100 rounded-2xl p-6 shadow-lg"
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
        >
          <h2 className="text-xl font-semibold text-gray-700 mb-2">
            Прогноз расходов
          </h2>
          {loadingForecast ? (
            <Loader2 className="w-6 h-6 mx-auto text-purple-700 animate-spin" />
          ) : errorForecast ? (
            <p className="text-red-500 text-sm">{errorForecast}</p>
          ) : spendingForecast ? (
            <div className="space-y-2">
              <p className="text-3xl font-bold text-purple-800">
                {spendingForecast.forecast.toLocaleString("ru-RU", {
                  style: "currency",
                  currency: "RUB",
                  maximumFractionDigits: 0,
                })}
              </p>
              <p className="text-sm text-gray-600">
                на {spendingForecast.next_month}
              </p>
              <div className="mt-3 pt-3 border-t border-purple-200 space-y-1">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Переменные расходы:</span>
                  <span className="font-semibold text-purple-700">
                    {spendingForecast.details.variable_spending.toLocaleString("ru-RU", {
                      style: "currency",
                      currency: "RUB",
                      maximumFractionDigits: 0,
                    })}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Фиксированные обязательства:</span>
                  <span className="font-semibold text-purple-700">
                    {spendingForecast.details.fixed_obligations.toLocaleString("ru-RU", {
                      style: "currency",
                      currency: "RUB",
                      maximumFractionDigits: 0,
                    })}
                  </span>
                </div>
              </div>
            </div>
          ) : (
            <p className="text-gray-500 text-sm">Нет данных</p>
          )}
        </motion.div>
      </div>

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
                        {ACCOUNT_SUBTYPE_RU[acc.account_subtype] || acc.account_subtype}
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

        {/* Рекомендации + Возможность покупки */}
        <motion.div
          className="bg-white rounded-2xl p-6 shadow-md space-y-6"
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
            <p className="text-center text-gray-600">Рекомендаций пока нет.</p>
          ) : (
            <div className="space-y-4">
              {displayedRecommendations.map((rec, i) => (
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
                    Категория: {rec.category} • {new Date(rec.created_at).toLocaleDateString("ru-RU")}
                  </p>
                </motion.div>
              ))}

              {/* Сообщение о подписке */}
              {!isSubscribed && (
                <motion.div
                  className="mt-4 p-4 rounded-xl border border-yellow-400 bg-yellow-50 flex flex-col items-center justify-center text-center space-y-3"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6 }}
                >
                  <div className="flex items-center gap-2 justify-center">
                    <PiggyBank className="w-6 h-6 text-yellow-600" />
                    <p className="font-semibold text-gray-800 text-lg">
                      Оформите подписку, чтобы увидеть все рекомендации!
                    </p>
                  </div>
                  <motion.button
                    onClick={handleSubscribe}
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    className="bg-blue-600 text-white px-5 py-2 rounded-lg hover:bg-blue-700 transition shadow-md"
                  >
                    Подписка 299₽/мес
                  </motion.button>
                </motion.div>
              )}
            </div>
          )}

          {/* 🔹 Блок проверки покупки */}
          <div className="mt-6 border-t pt-4">
            <h3 className="text-lg font-semibold text-gray-800 mb-2">
              Проверка покупки
            </h3>
            <div className="flex gap-2 items-center">
              <input
                type="number"
                min={0}
                value={purchaseAmount}
                onChange={(e) => setPurchaseAmount(e.target.value)}
                placeholder="Сумма покупки"
                className="w-full bg-gray-100 border border-gray-300 rounded-lg px-3 py-2"
              />
              <button
                onClick={checkAffordability}
                disabled={loadingAfford || !purchaseAmount || parseFloat(purchaseAmount) <= 0}
                className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition disabled:bg-blue-400"
              >
                Проверить
              </button>
            </div>
            {loadingAfford && (
              <div className="flex justify-center py-2">
                <Loader2 className="w-5 h-5 text-blue-600 animate-spin" />
              </div>
            )}
            {errorAfford && <p className="text-red-500 mt-2">{errorAfford}</p>}
            {affordability && (
              <div
                className={`mt-2 p-3 rounded-lg border ${
                  affordability.level === "CRITICAL"
                    ? "border-red-500 bg-red-50"
                    : affordability.level === "WARNING"
                    ? "border-yellow-400 bg-yellow-50"
                    : affordability.level === "CAUTION"
                    ? "border-orange-400 bg-orange-50"
                    : "border-green-400 bg-green-50"
                }`}
              >
                <p className={`font-semibold mb-2 ${
                  affordability.level === "CRITICAL"
                    ? "text-red-800"
                    : affordability.level === "WARNING"
                    ? "text-yellow-800"
                    : affordability.level === "CAUTION"
                    ? "text-orange-800"
                    : "text-green-800"
                }`}>
                  {affordability.message}
                </p>
                <p className="text-gray-700 whitespace-pre-line text-sm">{affordability.details}</p>
              </div>
            )}
          </div>
        </motion.div>
      </div>
    </motion.div>
  );
}
