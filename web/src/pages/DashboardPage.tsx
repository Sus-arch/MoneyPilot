import { useEffect, useState } from "react";
import { motion } from "framer-motion";

// Типы
interface Account {
  accountId: string;
  nickname: string;
  currency: string;
  balance: number;
}

interface Recommendation {
  id: number;
  title: string;
  description: string;
  category: string;
  priority: "low" | "medium" | "high";
  created_at: string;
}

// Тестовые данные
const TEST_ACCOUNTS: Account[] = [
  { accountId: "acc-1001", nickname: "Основной счёт", currency: "RUB", balance: 94691.8 },
  { accountId: "acc-1002", nickname: "Накопительный", currency: "RUB", balance: 50000.0 },
  { accountId: "acc-1003", nickname: "USD счёт", currency: "USD", balance: 1200.5 },
];

const TEST_RECOMMENDATIONS: Recommendation[] = [
  {
    id: 1,
    title: "Снизьте расходы на транспорт",
    description:
      "Вы тратите на такси на 30% больше среднего. Попробуйте использовать общественный транспорт или каршеринг.",
    category: "transport",
    priority: "medium",
    created_at: "2025-11-01T10:22:00Z",
  },
  {
    id: 2,
    title: "Инвестиции в депозиты",
    description:
      "У вас есть свободные 50 000 ₽ на счёте — откройте вклад под 9% годовых.",
    category: "savings",
    priority: "high",
    created_at: "2025-11-01T10:25:00Z",
  },
];

export default function DashboardPage() {

  const [accounts] = useState<Account[]>(TEST_ACCOUNTS);
  const [recommendations] = useState<Recommendation[]>(TEST_RECOMMENDATIONS);
  const [totalBalance, setTotalBalance] = useState<number>(0);

  useEffect(() => {
    const total = accounts
      .filter((acc) => acc.currency === "RUB")
      .reduce((sum, acc) => sum + acc.balance, 0);
    setTotalBalance(total);
  }, [accounts]);

  return (
    <motion.div
      className="max-w-6xl mx-auto p-8"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.6 }}
    >
      <motion.h1
        className="text-3xl font-bold text-center text-blue-700 mb-8"
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        Панель управления
      </motion.h1>

      {/* Общий баланс */}
      <motion.div
        className="bg-blue-100 rounded-2xl p-6 mb-8 text-center shadow-lg"
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="text-xl font-semibold text-gray-700 mb-2">Общий баланс</h2>
        <p className="text-4xl font-bold text-blue-800">
          {totalBalance.toLocaleString("ru-RU", {
            style: "currency",
            currency: "RUB",
          })}
        </p>
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Счета */}
        <motion.div
          className="bg-white rounded-2xl p-6 shadow-md"
          initial={{ x: -50, opacity: 0 }}
          animate={{ x: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
        >
          <h2 className="text-2xl font-semibold text-gray-800 mb-4">Ваши счета</h2>
          <div className="space-y-3">
            {accounts.map((acc, i) => (
              <motion.div
                key={acc.accountId}
                className="p-4 border rounded-xl bg-gray-50 hover:shadow-md transition flex justify-between items-center"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: i * 0.1 }}
              >
                <div>
                  <p className="font-semibold text-gray-800">{acc.nickname}</p>
                  <p className="text-sm text-gray-500">{acc.currency}</p>
                </div>
                <p className="text-lg font-bold text-blue-700">
                  {acc.balance.toLocaleString("ru-RU")} {acc.currency}
                </p>
              </motion.div>
            ))}
          </div>
        </motion.div>

        {/* Советы */}
        <motion.div
          className="bg-white rounded-2xl p-6 shadow-md"
          initial={{ x: 50, opacity: 0 }}
          animate={{ x: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          <h2 className="text-2xl font-semibold text-gray-800 mb-4">
            Персональные советы
          </h2>
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
        </motion.div>
      </div>
    </motion.div>
  );
}
