import React, { useState, useEffect, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useAuth } from "../context/AuthContext";
import { get, clearCache } from "../api/client";
import {
  CreditCard,
  Landmark,
  PiggyBank,
  Banknote,
  Wallet,
  Briefcase,
} from "lucide-react";

interface Account {
  account_id: string;
  nickname: string;
  status: string;
  currency: string;
  bank: string;
  account_subtype: string;
}

interface Balance {
  amount: string;
  currency: string;
  type: string;
  creditDebitIndicator: string;
  dateTime: string;
}

interface AccountDetail {
  account_id: string;
  account_type: string;
  account_subtype: string;
  description?: string;
  opening_date: string;
  balance?: Balance[];
}

const ACCOUNT_TYPE_TRANSLATIONS: Record<string, string> = {
  Checking: "Расчётный счёт",
  Savings: "Сберегательный счёт",
  Loan: "Кредитный счёт",
  Card: "Банковская карта",
  Deposit: "Депозит",
  Investment: "Инвестиционный счёт",
  Personal: "Личный счёт",
  Business: "Бизнес-счёт",
};

const ACCOUNT_ICONS: Record<string, React.ReactNode> = {
  Checking: <Landmark className="w-7 h-7 text-blue-600" />,
  Savings: <PiggyBank className="w-7 h-7 text-pink-500" />,
  Loan: <Banknote className="w-7 h-7 text-amber-600" />,
  Card: <CreditCard className="w-7 h-7 text-green-600" />,
  Deposit: <Wallet className="w-7 h-7 text-purple-600" />,
  Investment: <Briefcase className="w-7 h-7 text-indigo-500" />,
};

export default function AccountsPage() {
  const { token, currentBank, bankTokens } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<AccountDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [error, setError] = useState("");
  const fetchingRef = useRef(false);
  const lastBankRef = useRef<string | null>(null);

  const fetchAccounts = async () => {
    if (!token || !currentBank || !bankTokens[currentBank]) return;
    
    // Предотвращаем дублирующие запросы
    if (fetchingRef.current) return;
    if (lastBankRef.current === currentBank && accounts.length > 0) return;
    
    fetchingRef.current = true;
    setLoading(true);
    setError("");

    try {
      const res = await get("/accounts", {
        Authorization: `Bearer ${bankTokens[currentBank]}`,
      });

      const accountsData: Account[] = (res.accounts || []).map((a: any) => ({
        account_id: a.account_id,
        nickname: a.nickname,
        status: a.status,
        currency: a.currency,
        bank: a.bank,
        account_subtype: a.account_subtype,
      }));

      setAccounts(accountsData);
      lastBankRef.current = currentBank;
    } catch (err: any) {
      if (err.message?.includes("Rate limit")) {
        setError("Слишком много запросов. Подождите немного.");
      } else {
        setError("Не удалось получить список счетов");
      }
    } finally {
      setLoading(false);
      fetchingRef.current = false;
    }
  };

  useEffect(() => {
    // Очищаем кэш при смене банка
    if (lastBankRef.current && lastBankRef.current !== currentBank) {
      clearCache("/accounts");
      lastBankRef.current = null;
    }
    fetchAccounts();
  }, [currentBank]);

  const handleSelectAccount = async (account: Account) => {
    if (!token || !bankTokens[account.bank]) return;
    
    // Если уже загружаем этот же счет, не делаем повторный запрос
    if (loadingDetails && selectedAccount?.account_id === account.account_id) return;
    
    setLoadingDetails(true);
    setError("");
    setSelectedAccount(null);

    try {
      const [detailsRes, balanceRes] = await Promise.all([
        get(`/accounts/${account.account_id}/details`, {
          Authorization: `Bearer ${bankTokens[account.bank]}`,
          "X-Bank-Code": account.bank.toLowerCase(),
        }),
        get(`/accounts/${account.account_id}/balances`, {
          Authorization: `Bearer ${bankTokens[account.bank]}`,
          "X-Bank-Code": account.bank.toLowerCase(),
        }),
      ]);

      const details = detailsRes.data.account[0];
      const balances: Balance[] = (balanceRes.data?.balance || []).map((b: any) => ({
        amount: b.amount.amount,
        currency: b.amount.currency,
        type: b.type,
        creditDebitIndicator: b.creditDebitIndicator,
        dateTime: b.dateTime,
      }));

      setSelectedAccount({
        account_id: details.accountId,
        account_type: details.accountType,
        account_subtype: details.accountSubType,
        description: details.description,
        opening_date: details.openingDate,
        balance: balances,
      });
    } catch (err: any) {
      if (err.message?.includes("Rate limit")) {
        setError("Слишком много запросов. Подождите немного.");
      } else {
        setError("Не удалось получить данные по счёту");
      }
    } finally {
      setLoadingDetails(false);
    }
  };

  const groupedAccounts = accounts.reduce<Record<string, Account[]>>((acc, curr) => {
    acc[curr.bank] = acc[curr.bank] || [];
    acc[curr.bank].push(curr);
    return acc;
  }, {});

  const getAvailableBalance = (balances?: Balance[]) =>
    balances?.find((b) => b.type === "InterimAvailable");

  return (
    <motion.div
      className="max-w-6xl mx-auto p-8"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
    >
      <h1 className="text-3xl font-bold text-blue-700 mb-8 text-center">
        Ваши счета
      </h1>

      {loading && <p className="text-center text-gray-600">Загрузка счетов...</p>}
      {error && <p className="text-center text-red-500">{error}</p>}

      <div className="space-y-10">
        {Object.keys(groupedAccounts).map((bank) => (
          <motion.div
            key={bank}
            initial={{ y: 10, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
          >
            <h2 className="text-2xl font-semibold mb-4 text-gray-800">
              {bank.toUpperCase()}
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {groupedAccounts[bank].map((acc) => (
                <motion.div
                  key={acc.account_id}
                  className="bg-white rounded-2xl shadow-md hover:shadow-lg transition cursor-pointer border border-gray-100 p-5 flex items-center justify-between"
                  whileHover={{ scale: 1.02 }}
                  onClick={() => handleSelectAccount(acc)}
                >
                  <div className="flex items-center gap-4">
                    {ACCOUNT_ICONS[acc.account_subtype] || (
                      <Wallet className="w-7 h-7 text-gray-400" />
                    )}
                    <div>
                      <p className="font-semibold text-lg text-gray-800">
                        {ACCOUNT_TYPE_TRANSLATIONS[acc.account_subtype] || acc.nickname}
                      </p>
                      <p className="text-gray-500 text-sm">
                        {acc.status === "Enabled" ? "Активен" : "Не активен"} • {acc.currency}
                      </p>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          </motion.div>
        ))}
      </div>

      <AnimatePresence>
        {selectedAccount && (
          <motion.div
            className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <motion.div
              className="bg-white rounded-2xl shadow-2xl w-11/12 max-w-lg p-6 relative"
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
            >
              <button
                onClick={() => setSelectedAccount(null)}
                className="absolute top-3 right-4 text-gray-400 hover:text-gray-600 text-2xl"
              >
                &times;
              </button>

              <h2 className="text-2xl font-bold text-blue-700 mb-4">
                {ACCOUNT_TYPE_TRANSLATIONS[selectedAccount.account_subtype] ||
                  selectedAccount.account_id}
              </h2>

              <div className="space-y-2 text-gray-700">
                <p>
                  <span className="font-semibold">Тип:</span>{" "}
                  {selectedAccount.account_type === "Personal" ? "Личный" : "Бизнес"}
                </p>
                <p>
                  <span className="font-semibold">Подтип:</span>{" "}
                  {ACCOUNT_TYPE_TRANSLATIONS[selectedAccount.account_subtype] ||
                    selectedAccount.account_subtype}
                </p>
                {selectedAccount.description && (
                  <p>
                    <span className="font-semibold">Описание:</span>{" "}
                    {selectedAccount.description}
                  </p>
                )}
                <p>
                  <span className="font-semibold">Дата открытия:</span>{" "}
                  {selectedAccount.opening_date}
                </p>

                {loadingDetails && (
                  <p className="text-gray-600 mt-3">Загрузка баланса...</p>
                )}

                {selectedAccount.balance && (
                  <motion.div
                    className="mt-4 bg-blue-50 p-4 rounded-xl"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                  >
                    <p className="font-semibold text-gray-700 mb-1">Баланс:</p>
                    <p className="text-2xl font-bold text-blue-800">
                      {(() => {
                        const available = getAvailableBalance(selectedAccount.balance);
                        if (!available) return "-";
                        return `${available.amount} ${available.currency}`;
                      })()}
                    </p>
                  </motion.div>
                )}
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
