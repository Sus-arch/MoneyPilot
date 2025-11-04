import { useState, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import { get } from "../api/client";

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
  Card: "Карта",
  Deposit: "Депозит",
  Investment: "Инвестиционный счёт",
  Personal: "Личный счёт",
  Business: "Бизнес-счёт",
};

export default function AccountsPage() {
  const { token, currentBank, bankTokens } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<AccountDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [error, setError] = useState("");

  const fetchAccounts = async () => {
    if (!token || !currentBank || !bankTokens[currentBank]) return;
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
    } catch {
      setError("Не удалось получить список счетов");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, [token, currentBank, bankTokens]);

  const handleSelectAccount = async (account: Account) => {
    if (!token || !bankTokens[account.bank]) return;
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
    } catch {
      setError("Не удалось получить данные по счёту");
    } finally {
      setLoadingDetails(false);
    }
  };

  const groupedAccounts = accounts.reduce<Record<string, Account[]>>((acc, curr) => {
    acc[curr.bank] = acc[curr.bank] || [];
    acc[curr.bank].push(curr);
    return acc;
  }, {});

  const getAvailableBalance = (balances?: Balance[]) => {
    if (!balances) return null;
    return balances.find((b) => b.type === "InterimAvailable");
  };

  return (
    <div className="max-w-5xl mx-auto p-8">
      <h1 className="text-3xl font-bold text-blue-700 mb-6">Счета</h1>

      {loading && <p className="text-center text-gray-700">Загрузка счетов...</p>}
      {error && <p className="text-center text-red-500">{error}</p>}

      {Object.keys(groupedAccounts).map((bank) => (
        <div key={bank} className="mb-6">
          <h2 className="text-2xl font-semibold mb-4">{bank.toUpperCase()}</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
            {groupedAccounts[bank].map((acc) => (
              <div
                key={acc.account_id}
                className="p-4 border rounded-lg bg-gray-50 hover:bg-gray-100 cursor-pointer shadow-md transition"
                onClick={() => handleSelectAccount(acc)}
              >
                <p className="font-semibold text-lg">{ACCOUNT_TYPE_TRANSLATIONS[acc.account_subtype] || acc.nickname}</p>
                <p className="text-gray-600">
                  {acc.status === "Enabled" ? "Активен" : "Не активен"} • {acc.currency}
                </p>
              </div>
            ))}
          </div>
        </div>
      ))}

      {selectedAccount && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50">
          <div className="bg-white rounded-2xl shadow-xl w-11/12 max-w-lg p-6 relative">
            <button
              onClick={() => setSelectedAccount(null)}
              className="absolute top-4 right-4 text-gray-500 hover:text-gray-700 text-xl font-bold"
            >
              &times;
            </button>

            <h2 className="text-2xl font-bold text-blue-700 mb-4">
              {ACCOUNT_TYPE_TRANSLATIONS[selectedAccount.account_subtype] || selectedAccount.account_id}
            </h2>
            <div className="space-y-2 text-gray-700">
              <p>
                <span className="font-semibold">Тип:</span> {selectedAccount.account_type === "Personal" ? "Личный" : "Бизнес"}
              </p>
              <p>
                <span className="font-semibold">Подтип:</span> {ACCOUNT_TYPE_TRANSLATIONS[selectedAccount.account_subtype] || selectedAccount.account_subtype}
              </p>
              {selectedAccount.description && (
                <p>
                  <span className="font-semibold">Описание:</span> {selectedAccount.description}
                </p>
              )}
              <p>
                <span className="font-semibold">Дата открытия:</span> {selectedAccount.opening_date}
              </p>

              {loadingDetails && <p>Загрузка баланса...</p>}

              {selectedAccount.balance && (
                <div className="mt-2">
                  <p className="font-semibold">Баланс:</p>
                  <p className="text-lg font-semibold">
                    {(() => {
                      const available = getAvailableBalance(selectedAccount.balance);
                      if (!available) return "-";
                      return `${available.amount} ${available.currency}`;
                    })()}
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
