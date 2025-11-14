import { useState, useEffect, useMemo } from "react";
import { useAuth } from "../context/AuthContext";
import { get } from "../api/client";
import dayjs from "dayjs";
import { motion } from "framer-motion";

interface Account {
  accountId: string;
  nickname: string;
  status: string;
  currency: string;
  bank: string;
}

interface Transaction {
  transactionId: string;
  bookingDateTime: string;
  amount: string;
  currency: string;
  creditDebitIndicator: "credit" | "debit";
  transactionInformation: string;
  merchantName?: string;
  merchantCategory?: string;
  merchantAddress?: string;
  cardName?: string;
  status?: string;
}

const CATEGORY_TRANSLATIONS: Record<string, string> = {
  taxi: "🚕 Такси",
  cafe: "☕ Кафе",
  restaurant: "🍽 Ресторан",
  supermarket: "🛒 Супермаркет",
  entertainment: "🎬 Развлечения",
  transport: "🚌 Транспорт",
  utilities: "🏠 ЖКХ",
  salary: "💼 Зарплата",
  transfer: "💸 Перевод",
  clothing: "👔 Одежда",
  grocery: "🥖 Продукты",
  other: "📦 Прочее",
};

export default function TransactionsPage() {
  const { currentBank, bankTokens } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<{ bank: string; accountId: string } | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [fromDate, setFromDate] = useState(dayjs().subtract(1, "month").format("YYYY-MM-DD"));
  const [toDate, setToDate] = useState(dayjs().format("YYYY-MM-DD"));
  const [loading, setLoading] = useState(false);
  const [typeFilter, setTypeFilter] = useState<"all" | "credit" | "debit">("all");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");

  // --- Получаем счета ---
  const fetchAccounts = async () => {
    if (!currentBank || !bankTokens[currentBank]) return;
    const token = bankTokens[currentBank];
    if (!token) return;

    try {
      const res = await get("/accounts", {
        Authorization: `Bearer ${token}`,
      });

      const accountsData: Account[] = (res.accounts || []).map((a: any) => ({
        accountId: a.account_id,
        nickname: a.nickname || a.account_subtype || "Без имени",
        status: a.status,
        currency: a.currency,
        bank: a.bank,
      }));

      setAccounts(accountsData);

      if (accountsData.length > 0 && !selectedAccount) {
        const first = accountsData[0];
        setSelectedAccount({ bank: first.bank, accountId: first.accountId });
      }
    } catch (err) {
      console.error(err);
    }
  };

  // --- Получаем транзакции (берём то, что возвращает API) ---
  const fetchTransactions = async () => {
    if (!selectedAccount) return;
    if (!currentBank || !bankTokens[currentBank]) return;

    const { bank, accountId } = selectedAccount;
    setLoading(true);
    setTransactions([]);

    try {
      // Запрашиваем транзакции (API, похоже, возвращает всё — мы фильтруем локально)
      const res = await get(`/accounts/${accountId}/transactions`, {
        Authorization: `Bearer ${bankTokens[currentBank]}`,
        "X-Bank-Code": bank,
      });

      const rawTx = res.data?.transaction || [];

      // Нормализуем важные поля в нижний регистр для корректной фильтрации
      const mapped: Transaction[] = rawTx.map((tx: any) => ({
        transactionId: tx.transactionId,
        bookingDateTime: tx.bookingDateTime,
        amount: tx.amount?.amount ?? "0",
        currency: tx.amount?.currency ?? "RUB",
        creditDebitIndicator: (tx.creditDebitIndicator || "").toString().toLowerCase() === "credit" ? "credit" : "debit",
        transactionInformation: tx.transactionInformation || "-",
        merchantName: tx.merchant?.name || "—",
        merchantCategory: (tx.merchant?.category || "other").toString().toLowerCase(),
        merchantAddress: tx.merchant?.address || "",
        cardName: tx.card?.cardName || "—",
        status: tx.status || "",
      }));

      setTransactions(mapped);
    } catch (err) {
      console.error("Ошибка при загрузке транзакций:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentBank, bankTokens]);

  useEffect(() => {
    fetchTransactions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedAccount]);

  // --- Доступные категории (нижний регистр) ---
  const availableCategories = useMemo(() => {
    const allCats = transactions.map((t) => (t.merchantCategory || "other").toString().toLowerCase());
    const unique = Array.from(new Set(allCats));
    return ["all", ...unique];
  }, [transactions]);

  // --- Фильтрация (локально, по дате, типу, категории) ---
  const filteredTransactions = useMemo(() => {
    // нормализованные границы
    const from = fromDate ? dayjs(fromDate).startOf("day") : null;
    const to = toDate ? dayjs(toDate).endOf("day") : null;

    return transactions.filter((tx) => {
      // проверяем корректность даты транзакции
      const txMoment = tx.bookingDateTime ? dayjs(tx.bookingDateTime) : null;
      if (!txMoment || !txMoment.isValid()) return false;

      // фильтр по датам (включительно)
      if (from && txMoment.isBefore(from)) return false;
      if (to && txMoment.isAfter(to)) return false;

      // фильтр по типу (credit/debit)
      if (typeFilter !== "all" && tx.creditDebitIndicator.toLowerCase() !== typeFilter.toLowerCase()) {
        return false;
      }

      // фильтр по категории
      if (categoryFilter !== "all" && (tx.merchantCategory || "other").toLowerCase() !== categoryFilter.toLowerCase()) {
        return false;
      }

      return true;
    });
  }, [transactions, fromDate, toDate, typeFilter, categoryFilter]);

  return (
    <div className="max-w-6xl mx-auto p-8">
      <h1 className="text-3xl font-bold text-blue mb-6">Транзакции</h1>

      {/* Фильтры */}
      <div className="flex flex-col md:flex-row gap-4 mb-6">
        <select
          className="flex-1 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={selectedAccount ? `${selectedAccount.bank}_${selectedAccount.accountId}` : ""}
          onChange={(e) => {
            const [bank, accountId] = e.target.value.split("_");
            setSelectedAccount({ bank, accountId });
          }}
        >
          <option value="" disabled>
            Выберите счёт
          </option>
          {accounts.map((acc) => (
            <option key={`${acc.bank}_${acc.accountId}`} value={`${acc.bank}_${acc.accountId}`}>
              {acc.nickname} ({acc.currency}, {acc.bank.toUpperCase()})
            </option>
          ))}
        </select>

        <input
          type="date"
          className="bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={fromDate}
          onChange={(e) => setFromDate(e.target.value)}
        />
        <input
          type="date"
          className="bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={toDate}
          onChange={(e) => setToDate(e.target.value)}
        />

        <select
          className="bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value as any)}
        >
          <option value="all">Все операции</option>
          <option value="credit">Поступления</option>
          <option value="debit">Расходы</option>
        </select>

        <select
          className="bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value)}
        >
          {availableCategories.map((cat) => (
            <option key={cat} value={cat}>
              {cat === "all" ? "Все категории" : CATEGORY_TRANSLATIONS[cat] || cat}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <p className="text-blue">Загрузка...</p>
      ) : filteredTransactions.length === 0 ? (
        <p className="text-blue">Нет транзакций за выбранный период.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-gray-900 text-white rounded-lg shadow-lg">
            <thead className="bg-gray-800">
              <tr className="border-b border-gray-700">
                <th className="px-4 py-2 text-left">Дата</th>
                <th className="px-4 py-2 text-left">Описание</th>
                <th className="px-4 py-2 text-left">Магазин</th>
                <th className="px-4 py-2 text-left">Категория</th>
                <th className="px-4 py-2 text-left">Карта</th>
                <th className="px-4 py-2 text-left">Статус</th>
                <th className="px-4 py-2 text-left">Сумма</th>
              </tr>
            </thead>
            <tbody>
              {filteredTransactions.map((tx) => {
                const isCredit = tx.creditDebitIndicator === "credit";
                const catKey = (tx.merchantCategory || "other").toString().toLowerCase();
                const categoryName = CATEGORY_TRANSLATIONS[catKey] || catKey;

                return (
                  <motion.tr
                    key={tx.transactionId}
                    className="border-b border-gray-700 hover:bg-gray-800 transition"
                    whileHover={{ scale: 1.01 }}
                  >
                    <td className="px-4 py-2">{dayjs(tx.bookingDateTime).format("YYYY-MM-DD HH:mm")}</td>
                    <td className="px-4 py-2">{tx.transactionInformation}</td>
                    <td className="px-4 py-2">{tx.merchantName}</td>
                    <td className="px-4 py-2">{categoryName}</td>
                    <td className="px-4 py-2">{tx.cardName}</td>
                    <td className="px-4 py-2">{tx.status === "completed" ? "✅ Завершено" : "⏳ В обработке"}</td>
                    <td
                      className={`px-4 py-2 font-semibold ${isCredit ? "text-green-400" : "text-red-400"}`}
                    >
                      {isCredit ? "+" : "-"}
                      {tx.amount} {tx.currency}
                    </td>
                  </motion.tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
