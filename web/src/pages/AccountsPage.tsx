import { useState } from "react";

interface Account {
  accountId: string;
  nickname: string;
  status: string;
  currency: string;
  bank: string;
}

interface AccountDetail {
  accountId: string;
  accountType: string;
  accountSubType: string;
  description?: string;
  openingDate: string;
  balance?: {
    amount: string;
    currency: string;
  }[];
}

// Мок-данные
const MOCK_ACCOUNTS: Account[] = [
  { accountId: "acc-1901", nickname: "Checking счет", status: "Enabled", currency: "RUB", bank: "VBank" },
  { accountId: "acc-1905", nickname: "Checking счет", status: "Enabled", currency: "RUB", bank: "VBank" },
  { accountId: "acc-1902", nickname: "Savings счет", status: "Enabled", currency: "USD", bank: "ABank" },
  { accountId: "acc-1903", nickname: "Investment счет", status: "Disabled", currency: "EUR", bank: "SBank" },
];

const MOCK_ACCOUNT_DETAILS: Record<string, AccountDetail> = {
  "acc-1901": { accountId: "acc-1901", accountType: "Personal", accountSubType: "Checking", description: "Тестовый checking account", openingDate: "2024-10-30", balance: [{ amount: "94691.80", currency: "RUB" }, { amount: "500.00", currency: "USD" }] },
  "acc-1905": { accountId: "acc-1905", accountType: "Personal", accountSubType: "Checking", description: "Тестовый checking account", openingDate: "2024-10-30", balance: [{ amount: "94691.80", currency: "RUB" }, { amount: "500.00", currency: "USD" }] },
  "acc-1902": { accountId: "acc-1902", accountType: "Personal", accountSubType: "Savings", openingDate: "2024-11-01", balance: [{ amount: "1200.50", currency: "USD" }] },
  "acc-1903": { accountId: "acc-1903", accountType: "Investment", accountSubType: "Brokerage", openingDate: "2025-01-15", balance: [{ amount: "10000.00", currency: "EUR" }] },
};

export default function AccountsPageModal() {
  const [accounts] = useState<Account[]>(MOCK_ACCOUNTS);
  const [selectedAccount, setSelectedAccount] = useState<AccountDetail | null>(null);

  const groupedAccounts = accounts.reduce<Record<string, Account[]>>((acc, curr) => {
    acc[curr.bank] = acc[curr.bank] || [];
    acc[curr.bank].push(curr);
    return acc;
  }, {});

  return (
    <div className="max-w-5xl mx-auto p-8">
      <h1 className="text-3xl font-bold text-blue-700 mb-6">Счета (тестовые данные)</h1>

      {Object.keys(groupedAccounts).map((bank) => (
        <div key={bank} className="mb-6">
          <h2 className="text-2xl font-semibold mb-4">{bank}</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
            {groupedAccounts[bank].map((acc) => (
              <div
                key={acc.accountId}
                className="p-4 border rounded-lg bg-gray-50 hover:bg-gray-100 cursor-pointer shadow-sm transition"
                onClick={() => setSelectedAccount(MOCK_ACCOUNT_DETAILS[acc.accountId])}
              >
                <p className="font-semibold text-lg">{acc.nickname}</p>
                <p className="text-gray-600">{acc.status} • {acc.currency}</p>
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

            <h2 className="text-2xl font-bold text-blue-700 mb-4">{selectedAccount.accountId}</h2>
            <div className="space-y-2 text-gray-700">
              <p><span className="font-semibold">Тип:</span> {selectedAccount.accountType}</p>
              <p><span className="font-semibold">Подтип:</span> {selectedAccount.accountSubType}</p>
              {selectedAccount.description && <p><span className="font-semibold">Описание:</span> {selectedAccount.description}</p>}
              <p><span className="font-semibold">Дата открытия:</span> {selectedAccount.openingDate}</p>
              {selectedAccount.balance && (
                <div className="mt-2">
                  <p className="font-semibold">Баланс:</p>
                  {selectedAccount.balance.map((b, i) => (
                    <p key={i}>{b.amount} {b.currency}</p>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
