import { useState, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import { get } from "../api/client";
import dayjs from "dayjs";

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
  creditDebitIndicator: "Credit" | "Debit";
  transactionInformation: string;
}

const ACCOUNT_TYPE_TRANSLATIONS: Record<string, string> = {
  Checking: "Расчётный счёт",
  Savings: "Сберегательный счёт",
  Loan: "Кредитный счёт",
  Card: "Карта",
  Deposit: "Депозит",
  Investment: "Инвестиционный счёт",
  Personal: "Личный счёт",
  Business: "Бизнес-счёт",
};

export default function TransactionsPage() {
  const { currentBank, bankTokens } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<{ bank: string; accountId: string } | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [fromDate, setFromDate] = useState(dayjs().subtract(1, "month").format("YYYY-MM-DD"));
  const [toDate, setToDate] = useState(dayjs().format("YYYY-MM-DD"));
  const [loading, setLoading] = useState(false);

  // --- Фильтры ---
  const [typeFilter, setTypeFilter] = useState<"all" | "credit" | "debit">("all");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");

  // Возможные категории
  const categories = [
    "all", 
    "💼 Зарплата", 
    "💰 Подработка",
    "🎬 Развлечения", 
    "🚌 Транспорт", 
    "🏠 ЖКХ",
    "🏪 Продукты",
  ];

  // --- Получаем все счета ---
    const fetchAccounts = async () => {
    if (!currentBank) return;
    const token = bankTokens[currentBank];
    if (!token) return;

    try {
      const res = await get("/accounts", {
        Authorization: `Bearer ${token}`,
      });

      const accountsData: Account[] = (res.accounts || []).map((a: any) => {
        console.log(a.account_subtype);
        const translatedType =
          ACCOUNT_TYPE_TRANSLATIONS[a.account_subtype] ||
          "Неизвестный тип";

        return {
          accountId: a.account_id,
          nickname: `${translatedType || a.nickname}`,
          status: a.status,
          currency: a.currency,
          bank: a.bank,
        };
      });

      setAccounts(accountsData);

      if (accountsData.length > 0 && !selectedAccount) {
        const first = accountsData[0];
        setSelectedAccount({ bank: first.bank, accountId: first.accountId });
      }
    } catch (err) {
      console.error(err);
    }
  };


  // --- Получаем транзакции ---
  const fetchTransactions = async () => {
    if (!selectedAccount) return;
    const { bank, accountId } = selectedAccount;
    const token = bankTokens[currentBank || bank];
    if (!token) return;

    setLoading(true);
    setTransactions([]);

    try {
      const query = new URLSearchParams({
        from: `${fromDate}T00:00:00Z`,
        to: `${toDate}T23:59:59Z`,
        page: "0",
        limit: "100",
      }).toString();

      const res = await get(`/accounts/${accountId}/transactions?${query}`, {
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": bank,
      });

      const rawTx = res.data?.transaction || [];
      const from = dayjs(fromDate).startOf("day");
      const to = dayjs(toDate).endOf("day");

      const allTx: Transaction[] = rawTx
        .filter((tx: any) => {
          const txDate = dayjs(tx.bookingDateTime);
          return txDate.isValid() && txDate.isAfter(from.subtract(1, "ms")) && txDate.isBefore(to.add(1, "ms"));
        })
        .map((tx: any) => ({
          transactionId: tx.transactionId,
          bookingDateTime: tx.bookingDateTime,
          amount: tx.amount.amount,
          currency: tx.amount.currency,
          creditDebitIndicator: tx.creditDebitIndicator,
          transactionInformation: tx.transactionInformation || "-",
        }));

      setTransactions(allTx);
    } catch (err) {
      console.error("Ошибка при загрузке транзакций:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, [currentBank, bankTokens]);

  useEffect(() => {
    fetchTransactions();
  }, [selectedAccount, fromDate, toDate]);

  // --- Фильтрация транзакций ---
  const filteredTransactions = transactions.filter((tx) => {
    const matchType =
      typeFilter === "all" ||
      (typeFilter === "credit" && tx.creditDebitIndicator === "Credit") ||
      (typeFilter === "debit" && tx.creditDebitIndicator === "Debit");

    const matchCategory =
      categoryFilter === "all" ||
      tx.transactionInformation.toLowerCase().includes(
        categoryFilter
          .replace(/[^\p{L}\p{N}]+/gu, "")
          .toLowerCase()
      );

    return matchType && matchCategory;
  });

  return (
    <div className="max-w-5xl mx-auto p-8">
      <h1 className="text-3xl font-bold text-blue mb-6">Транзакции</h1>

      {/* Фильтры */}
      <div className="flex flex-col sm:flex-row gap-4 mb-6">
        <select
          className="w-full sm:w-1/3 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={selectedAccount ? `${selectedAccount.bank}_${selectedAccount.accountId}` : ""}
          onChange={(e) => {
            const [bank, accountId] = e.target.value.split("_");
            setSelectedAccount({ bank, accountId });
          }}
        >
          <option value="" disabled>
            Выберите счет
          </option>
          {accounts.map((acc) => (
            <option key={`${acc.bank}_${acc.accountId}`} value={`${acc.bank}_${acc.accountId}`}>
              {acc.nickname} {acc.currency} ({acc.bank.toUpperCase()})
            </option>
          ))}
        </select>

        <input
          type="date"
          className="w-full sm:w-1/4 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={fromDate}
          onChange={(e) => setFromDate(e.target.value)}
        />
        <input
          type="date"
          className="w-full sm:w-1/4 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={toDate}
          onChange={(e) => setToDate(e.target.value)}
        />

        <select
          className="w-full sm:w-1/4 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value as "all" | "credit" | "debit")}
        >
          <option value="all">Все операции</option>
          <option value="credit">Поступления</option>
          <option value="debit">Расходы</option>
        </select>

        <select
          className="w-full sm:w-1/4 bg-gray-800 text-white border border-gray-600 rounded-lg px-4 py-2"
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value)}
        >
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat === "all" ? "Все категории" : cat}
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
          <table className="min-w-full bg-gray-900 text-white rounded-lg">
            <thead>
              <tr className="border-b border-gray-700">
                <th className="px-4 py-2 text-left">Дата</th>
                <th className="px-4 py-2 text-left">Сумма</th>
                <th className="px-4 py-2 text-left">Валюта</th>
                <th className="px-4 py-2 text-left">Описание</th>
              </tr>
            </thead>
            <tbody>
              {filteredTransactions.map((tx) => {
                const isCredit = tx.creditDebitIndicator === "Credit";
                return (
                  <tr key={tx.transactionId} className="border-b border-gray-700 hover:bg-gray-800 transition">
                    <td className="px-4 py-2">{dayjs(tx.bookingDateTime).format("YYYY-MM-DD")}</td>
                    <td
                      className={`px-4 py-2 font-semibold ${
                        isCredit ? "text-green-400" : "text-red-400"
                      }`}
                    >
                      {isCredit ? "+" : "-"}
                      {tx.amount}
                    </td>
                    <td className="px-4 py-2">{tx.currency}</td>
                    <td className="px-4 py-2">{tx.transactionInformation}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
