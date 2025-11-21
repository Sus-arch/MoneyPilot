import { useState, useEffect, useMemo, useRef } from "react";
import { useAuth } from "../context/AuthContext";
import { get, debouncedGet } from "../api/client";
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

  const fetchingAccountsRef = useRef(false);
  const fetchingTransactionsRef = useRef(false);
  const lastAccountRef = useRef<string | null>(null);

  // --- Получаем счета ---
  const fetchAccounts = async () => {
    if (!currentBank) return;
    const token = bankTokens[currentBank];
    if (!token) return;

    if (fetchingAccountsRef.current) return;
    fetchingAccountsRef.current = true;

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
    } catch (err: any) {
      console.error(err);
    } finally {
      fetchingAccountsRef.current = false;
    }
  };

  // --- Получаем транзакции ---
  const fetchTransactions = async () => {
    if (!selectedAccount) return;
    const { bank, accountId } = selectedAccount;
    const token = bankTokens[currentBank || bank];
    if (!token) return;

    const accountKey = `${bank}_${accountId}_${fromDate}_${toDate}`;

    if (fetchingTransactionsRef.current && lastAccountRef.current === accountKey) return;

    fetchingTransactionsRef.current = true;
    lastAccountRef.current = accountKey;
    setLoading(true);
    setTransactions([]);

    try {
      const query = new URLSearchParams({
        from: `${fromDate}T00:00:00Z`,
        to: `${toDate}T23:59:59Z`,
        page: "0",
        limit: "100",
      }).toString();

      const res = await debouncedGet(`/accounts/${accountId}/transactions?${query}`, {
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": bank,
      }, 500);

      const rawTx = res.data?.transaction || [];

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
    } catch (err: any) {
      console.error("Ошибка при загрузке транзакций:", err);
      if (err.message?.includes("Rate limit")) {
        setTransactions([]);
      }
    } finally {
      setLoading(false);
      fetchingTransactionsRef.current = false;
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, [currentBank]);

  useEffect(() => {
    const timer = setTimeout(() => {
      fetchTransactions();
    }, 500);

    return () => clearTimeout(timer);
  }, [selectedAccount, fromDate, toDate]);

  // --- Доступные категории ---
  const availableCategories = useMemo(() => {
    const allCats = transactions.map((t) => (t.merchantCategory || "other").toString().toLowerCase());
    const unique = Array.from(new Set(allCats));
    return ["all", ...unique];
  }, [transactions]);

  // --- Фильтрация транзакций ---
  const filteredTransactions = useMemo(() => {
    const from = fromDate ? dayjs(fromDate).startOf("day") : null;
    const to = toDate ? dayjs(toDate).endOf("day") : null;

    return transactions.filter((tx) => {
      const txMoment = tx.bookingDateTime ? dayjs(tx.bookingDateTime) : null;
      if (!txMoment || !txMoment.isValid()) return false;

      if (from && txMoment.isBefore(from)) return false;
      if (to && txMoment.isAfter(to)) return false;

      const type = (tx.creditDebitIndicator || "").toString().toLowerCase();
      if (typeFilter !== "all" && type !== typeFilter.toLowerCase()) return false;

      const cat = (tx.merchantCategory || "other").toString().toLowerCase();
      if (categoryFilter !== "all" && cat !== categoryFilter.toLowerCase()) return false;

      return true;
    });
  }, [transactions, fromDate, toDate, typeFilter, categoryFilter]);

  return (
    <div className="max-w-6xl mx-auto p-4 md:p-6 lg:p-8">
      <h1 className="text-2xl md:text-3xl font-bold text-blue mb-4 md:mb-6">Транзакции</h1>

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
          onChange={(e) => setTypeFilter(e.target.value as "all" | "credit" | "debit")}
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
        <>
          {/* Desktop Table View */}
          <div className="hidden lg:block overflow-x-auto">
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
                  const categoryName = CATEGORY_TRANSLATIONS[tx.merchantCategory || "other"] || tx.merchantCategory;
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
                      <td className={`px-4 py-2 font-semibold ${isCredit ? "text-green-400" : "text-red-400"}`}>
                        {isCredit ? "+" : "-"}
                        {tx.amount} {tx.currency}
                      </td>
                    </motion.tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Mobile Card View */}
          <div className="lg:hidden space-y-4">
            {filteredTransactions.map((tx) => {
              const isCredit = tx.creditDebitIndicator === "credit";
              const categoryName = CATEGORY_TRANSLATIONS[tx.merchantCategory || "other"] || tx.merchantCategory;
              return (
                <motion.div
                  key={tx.transactionId}
                  className="bg-gray-900 text-white rounded-lg p-4 shadow-lg"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.2 }}
                >
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex-1">
                      <p className="font-semibold text-sm text-gray-300">
                        {dayjs(tx.bookingDateTime).format("DD.MM.YYYY HH:mm")}
                      </p>
                      <p className="text-white font-medium mt-1">{tx.transactionInformation}</p>
                    </div>
                    <span className={`text-lg font-bold ${isCredit ? "text-green-400" : "text-red-400"}`}>
                      {isCredit ? "+" : "-"}
                      {tx.amount} {tx.currency}
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-sm text-gray-400 mt-3 pt-3 border-t border-gray-700">
                    <div>
                      <span className="text-gray-500">Магазин:</span>
                      <p className="text-white">{tx.merchantName}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Категория:</span>
                      <p className="text-white">{categoryName}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Карта:</span>
                      <p className="text-white">{tx.cardName}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Статус:</span>
                      <p className="text-white">{tx.status === "completed" ? "✅ Завершено" : "⏳ В обработке"}</p>
                    </div>
                  </div>
                </motion.div>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}
