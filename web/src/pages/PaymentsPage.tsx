import { useState, useRef, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import { post, get } from "../api/client";
import {
  Loader2,
  CheckCircle,
  XCircle,
  AlertCircle,
  CreditCard,
  Wallet,
  Banknote,
} from "lucide-react";

interface Payment {
  paymentId: string;
  status: string;
  amount: string;
  currency: string;
  creationDateTime?: string;
  statusUpdateDateTime?: string;
  description?: string;
}

const BANKS = [
  { id: "vbank", name: "VBank", icon: <Banknote className="w-6 h-6 text-blue-600" /> },
  { id: "abank", name: "ABank", icon: <CreditCard className="w-6 h-6 text-green-600" /> },
  { id: "sbank", name: "SBank", icon: <Wallet className="w-6 h-6 text-purple-600" /> },
];

const STATUS_COLORS: Record<string, string> = {
  pending: "text-yellow-600 bg-yellow-50 border-yellow-200",
  accepted: "text-blue-600 bg-blue-50 border-blue-200",
  completed: "text-green-600 bg-green-50 border-green-200",
  rejected: "text-red-600 bg-red-50 border-red-200",
  failed: "text-red-600 bg-red-50 border-red-200",
};

const STATUS_RU: Record<string, string> = {
  pending: "В обработке",
  accepted: "Принят",
  completed: "Завершен",
  rejected: "Отклонен",
  failed: "Ошибка",
};

interface Account {
  account_id: string;
  nickname: string;
  status: string;
  currency: string;
  bank: string;
  account_subtype: string;
  identification?: string;
}

export default function PaymentsPage() {
  const { currentBank, bankTokens } = useAuth();
  const [selectedBank, setSelectedBank] = useState<string | null>(currentBank || null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [useDebtorFromList, setUseDebtorFromList] = useState(false);
  const [paymentData, setPaymentData] = useState({
    debtor_account: "",
    creditor_account: "",
    amount: "",
    currency: "RUB",
    comment: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [payment, setPayment] = useState<Payment | null>(null);
  const [showConsentModal, setShowConsentModal] = useState(false);
  const [consentLoading, setConsentLoading] = useState(false);
  const [consentError, setConsentError] = useState<string | null>(null);
  const [consentResult, setConsentResult] = useState<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const activeBankRef = useRef<string | null>(null);
  const activeConsentIdRef = useRef<string | null>(null);
  const activeRequestIdRef = useRef<string | null>(null);
  const paymentStatusIntervalRef = useRef<number | null>(null);

  // Загрузка счетов
  const fetchAccounts = async (bankId: string) => {
    const token = bankTokens[bankId];
    if (!token) return;

    setLoadingAccounts(true);
    try {
      const connectedBanks = Object.keys(bankTokens).filter((bank) => bankTokens[bank]);
      const bankCodesHeader = connectedBanks.join(",");

      const res = await get("/accounts", {
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": bankCodesHeader,
      });

      const accountsData: Account[] = (res.accounts || []).map((a: any) => ({
        account_id: a.account_id,
        nickname: a.nickname,
        status: a.status,
        currency: a.currency,
        bank: a.bank,
        account_subtype: a.account_subtype,
        identification: a.identification,
      }));

      setAccounts(accountsData);
    } catch (err: any) {
      console.error("Ошибка загрузки счетов:", err);
    } finally {
      setLoadingAccounts(false);
    }
  };

  useEffect(() => {
    if (selectedBank && bankTokens[selectedBank]) {
      fetchAccounts(selectedBank);
    }
  }, [selectedBank]);

  // Проверка наличия согласия на перевод
  const checkPaymentConsent = async (bankId: string, debtorAccountId: string, creditorAccount: string, amount: number) => {
    const token = bankTokens[bankId];
    if (!token) return null;

    try {
      // Используем введенный идентификатор счета напрямую
      const debtorIdentification = debtorAccountId;
      console.log(`Используем идентификатор для проверки согласия: ${debtorIdentification}`);

      // Пытаемся создать согласие single_use - если оно уже существует, API вернет его
      const response = await post(
        "/payment-consents/request",
        {
          consent_type: "single_use",
          debtor_account: debtorIdentification,
          creditor_account: creditorAccount,
          amount: amount,
          currency: paymentData.currency,
          reference: paymentData.comment || undefined,
        },
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": bankId,
        }
      );

      // Если согласие уже существует и активно
      if (response.message === "active consent exists" && response.status === "approved") {
        return response.consent_id || response.request_id;
      }

      // Если согласие создано и одобрено сразу
      if (response.status === "approved" || response.auto_approved) {
        return response.consent_id || response.request_id;
      }

      // Если требуется подтверждение - возвращаем null, чтобы показать модальное окно
      if (response.status === "pending" || !response.auto_approved) {
        return null;
      }

      return null;
    } catch (err: any) {
      console.error("Ошибка проверки согласия:", err);
      // Если ошибка - возвращаем null, чтобы показать модальное окно
      return null;
    }
  };

  // Подключение WebSocket для ожидания согласия
  const connectWebSocket = (bankId: string, consentId?: string, requestId?: string) => {
    if (wsRef.current) wsRef.current.close();

    const socket = new WebSocket("ws://localhost:8080/ws");
    wsRef.current = socket;
    activeBankRef.current = bankId;
    if (consentId) activeConsentIdRef.current = consentId;
    if (requestId) activeRequestIdRef.current = requestId;

    socket.onopen = () => console.log(`✅ WebSocket для ${bankId} открыт`);
    socket.onclose = () => {
      console.log(`❌ WebSocket закрыт для ${bankId}`);
      wsRef.current = null;
      activeBankRef.current = null;
      activeConsentIdRef.current = null;
      activeRequestIdRef.current = null;
    };

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        console.log("📨 WebSocket message для согласия:", msg);

        if (!msg.consent_id || !msg.status) return;

        const currentBank = activeBankRef.current;
        if (!currentBank) return;

        // Проверяем, относится ли сообщение к нашему согласию
        if (activeConsentIdRef.current || activeRequestIdRef.current) {
          const matchesConsentId = activeConsentIdRef.current && msg.consent_id === activeConsentIdRef.current;
          const matchesRequestId = activeRequestIdRef.current && msg.consent_id === activeRequestIdRef.current;

          const isPaymentConsent = true;
          const isApprovedStatus = msg.status === "approved";

          if (!matchesConsentId && !matchesRequestId) {
            if (isPaymentConsent && isApprovedStatus) {
              console.log(`✅ Принимаем обновленный consent_id: ${msg.consent_id}`);
            } else {
              return;
            }
          }
        }

        if (msg.status === "approved") {
          // Обновляем activeConsentIdRef на актуальный consent_id из WebSocket сообщения
          if (msg.consent_id) {
            activeConsentIdRef.current = msg.consent_id;
            // Очищаем request_id, так как теперь есть consent_id
            activeRequestIdRef.current = null;
          }
          // Обновляем consentResult с актуальным consent_id из WebSocket сообщения
          // Важно: после подтверждения request_id заменяется на consent_id
          setConsentResult({ 
            ...msg, 
            status: "approved", 
            consent_id: msg.consent_id,
            // Удаляем request_id, так как он больше недействителен
            request_id: undefined
          });
          setConsentError(null);
          // Закрываем модальное окно и продолжаем создание платежа
          setShowConsentModal(false);
          createPaymentAfterConsent();
          socket.close();
        }
      } catch (err) {
        console.error("Ошибка при обработке WebSocket-сообщения:", err);
      }
    };
  };

  // Создание согласия на перевод
  const handleCreateConsent = async () => {
    if (!selectedBank) {
      setConsentError("Выберите банк");
      return;
    }

    const token = bankTokens[selectedBank];
    if (!token) {
      setConsentError("Банк не подключен");
      return;
    }

    if (!paymentData.debtor_account || !paymentData.creditor_account || !paymentData.amount) {
      setConsentError("Заполните все обязательные поля");
      return;
    }

    setConsentError(null);
    setConsentResult(null);
    setConsentLoading(true);

    try {
      const amount = parseFloat(paymentData.amount);
      if (isNaN(amount) || amount <= 0) {
        setConsentError("Укажите корректную сумму");
        return;
      }

      // Используем введенный идентификатор счета напрямую
      const debtorIdentification = paymentData.debtor_account;
      console.log(`Используем идентификатор для согласия: ${debtorIdentification}`);

      const response = await post(
        "/payment-consents/request",
        {
          consent_type: "single_use",
          debtor_account: debtorIdentification,
          creditor_account: paymentData.creditor_account,
          amount: amount,
          currency: paymentData.currency,
          reference: paymentData.comment || undefined,
        },
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": selectedBank,
        }
      );

      setConsentResult(response);

      if (response.status === "approved") {
        // Согласие одобрено сразу - закрываем модальное окно и создаем платеж
        setShowConsentModal(false);
        createPaymentAfterConsent();
      } else if (!response.auto_approved) {
        // Требуется подтверждение - ждем через WebSocket
        connectWebSocket(selectedBank, response.consent_id, response.request_id);
      }
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setConsentError("Слишком много запросов. Подождите немного.");
      } else {
        setConsentError("Ошибка при создании согласия. Попробуйте позже.");
      }
    } finally {
      setConsentLoading(false);
    }
  };

  // Создание платежа после получения согласия
  const createPaymentAfterConsent = async () => {
    if (!selectedBank) return;

    const token = bankTokens[selectedBank];
    if (!token) return;

    setLoading(true);
    setError(null);

    try {
      const amount = parseFloat(paymentData.amount);
      if (isNaN(amount) || amount <= 0) {
        setError("Укажите корректную сумму");
        setLoading(false);
        return;
      }

      // Приоритетно используем consent_id из WebSocket сообщения (после подтверждения)
      // Важно: после подтверждения через WebSocket request_id заменяется на consent_id
      // Поэтому используем только consent_id, request_id недействителен после подтверждения
      let consentId = consentResult?.consent_id || activeConsentIdRef.current;
      
      // Если consent_id еще нет (согласие было одобрено сразу при создании), используем из результата
      if (!consentId) {
        consentId = consentResult?.consent_id || consentResult?.request_id || activeRequestIdRef.current;
      }

      if (!consentId) {
        setError("Не удалось получить ID согласия");
        setLoading(false);
        return;
      }

      console.log(`Используем consent_id для платежа: ${consentId} (из consentResult.consent_id: ${consentResult?.consent_id}, из activeConsentIdRef: ${activeConsentIdRef.current}, из consentResult.request_id: ${consentResult?.request_id})`);

      // Используем введенный идентификатор счета напрямую
      const debtorIdentification = paymentData.debtor_account;
      console.log(`Используем идентификатор для платежа после согласия: ${debtorIdentification}`);

      const response = await post(
        "/payments",
        {
          data: {
            initiation: {
              instructedAmount: {
                amount: paymentData.amount,
                currency: paymentData.currency,
              },
              debtorAccount: {
                schemeName: "OBAN",
                identification: debtorIdentification,
              },
              creditorAccount: {
                schemeName: "OBAN",
                identification: paymentData.creditor_account,
              },
              comment: paymentData.comment || undefined,
            },
          },
        },
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": selectedBank,
          "X-Payment-Consent-Id": consentId,
        }
      );

      if (response.data) {
        setPayment({
          paymentId: response.data.paymentId,
          status: response.data.status,
          amount: response.data.amount,
          currency: response.data.currency,
          creationDateTime: response.data.creationDateTime,
          statusUpdateDateTime: response.data.statusUpdateDateTime,
          description: response.data.description,
        });

        // Запускаем проверку статуса платежа
        if (response.data.paymentId) {
          startPaymentStatusPolling(response.data.paymentId);
        }
      }
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setError("Слишком много запросов. Подождите немного.");
      } else if (err.message?.includes("payment consent")) {
        setError("Требуется согласие на перевод. Создайте согласие сначала.");
      } else {
        setError("Ошибка при создании платежа. Попробуйте позже.");
      }
    } finally {
      setLoading(false);
    }
  };

  // Создание платежа
  const handleCreatePayment = async () => {
    if (!selectedBank) {
      setError("Выберите банк");
      return;
    }

    const token = bankTokens[selectedBank];
    if (!token) {
      setError("Банк не подключен");
      return;
    }

    if (!paymentData.debtor_account || !paymentData.creditor_account || !paymentData.amount) {
      setError("Заполните все обязательные поля");
      return;
    }

    const amount = parseFloat(paymentData.amount);
    if (isNaN(amount) || amount <= 0) {
      setError("Укажите корректную сумму");
      return;
    }

    // Проверяем наличие согласия
    const consentId = await checkPaymentConsent(selectedBank, paymentData.debtor_account, paymentData.creditor_account, amount);

    if (consentId === null) {
      // Согласия нет или требуется подтверждение - показываем модальное окно
      setShowConsentModal(true);
      return;
    }

    // Согласие есть - создаем платеж
    setError(null);
    setLoading(true);

    try {
      // Используем введенный идентификатор счета напрямую
      const debtorIdentification = paymentData.debtor_account;
      console.log(`Используем идентификатор для платежа: ${debtorIdentification}`);

      const response = await post(
        "/payments",
        {
          data: {
            initiation: {
              instructedAmount: {
                amount: paymentData.amount,
                currency: paymentData.currency,
              },
              debtorAccount: {
                schemeName: "OBAN",
                identification: debtorIdentification,
              },
              creditorAccount: {
                schemeName: "OBAN",
                identification: paymentData.creditor_account,
              },
              comment: paymentData.comment || undefined,
            },
          },
        },
        {
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": selectedBank,
          "X-Payment-Consent-Id": consentId,
        }
      );

      if (response.data) {
        setPayment({
          paymentId: response.data.paymentId,
          status: response.data.status,
          amount: response.data.amount,
          currency: response.data.currency,
          creationDateTime: response.data.creationDateTime,
          statusUpdateDateTime: response.data.statusUpdateDateTime,
          description: response.data.description,
        });

        // Запускаем проверку статуса платежа
        if (response.data.paymentId) {
          startPaymentStatusPolling(response.data.paymentId);
        }
      }
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setError("Слишком много запросов. Подождите немного.");
      } else if (err.message?.includes("payment consent")) {
        // Если требуется согласие - показываем модальное окно
        setShowConsentModal(true);
      } else {
        setError("Ошибка при создании платежа. Попробуйте позже.");
      }
    } finally {
      setLoading(false);
    }
  };

  // Проверка статуса платежа
  const checkPaymentStatus = async (paymentId: string) => {
    if (!selectedBank) return;

    const token = bankTokens[selectedBank];
    if (!token) return;

    try {
      const response = await get(`/payments/${paymentId}`, {
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": selectedBank,
      });

      if (response.data) {
        setPayment((prev) => ({
          ...prev!,
          status: response.data.status,
          statusUpdateDateTime: response.data.statusUpdateDateTime,
        }));

        // Если платеж завершен или отклонен - останавливаем проверку
        if (response.data.status === "completed" || response.data.status === "rejected" || response.data.status === "failed") {
          if (paymentStatusIntervalRef.current) {
            clearInterval(paymentStatusIntervalRef.current);
            paymentStatusIntervalRef.current = null;
          }
        }
      }
    } catch (err) {
      console.error("Ошибка проверки статуса платежа:", err);
    }
  };

  // Запуск периодической проверки статуса платежа
  const startPaymentStatusPolling = (paymentId: string) => {
    // Проверяем сразу
    checkPaymentStatus(paymentId);

    // Затем каждые 3 секунды
    if (paymentStatusIntervalRef.current) {
      clearInterval(paymentStatusIntervalRef.current);
    }
    paymentStatusIntervalRef.current = setInterval(() => {
      checkPaymentStatus(paymentId);
    }, 3000);
  };

  useEffect(() => {
    return () => {
      if (paymentStatusIntervalRef.current) {
        clearInterval(paymentStatusIntervalRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const connectedBanks = BANKS.filter((b) => bankTokens[b.id]);

  return (
    <div className="max-w-4xl mx-auto mt-10 p-6">
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-8">
        Создание платежа
      </h1>

      {connectedBanks.length === 0 ? (
        <div className="text-center py-12 bg-yellow-50 border border-yellow-200 rounded-lg">
          <AlertCircle className="w-12 h-12 text-yellow-600 mx-auto mb-4" />
          <p className="text-gray-700">
            Нет подключённых банков. Подключите банк на странице "Банки", чтобы создать платеж.
          </p>
        </div>
      ) : (
        <>
          {/* Выбор банка */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-white mb-2">
              Выберите банк
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              {connectedBanks.map((bank) => (
                <button
                  key={bank.id}
                  onClick={() => setSelectedBank(bank.id)}
                  className={`p-4 rounded-lg border-2 transition ${
                    selectedBank === bank.id
                      ? "border-blue-500 bg-blue-50"
                      : "border-gray-300 bg-white hover:border-gray-400"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {bank.icon}
                    <span className="font-medium">{bank.name}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {selectedBank && (
            <div className="bg-gray-800 rounded-2xl shadow-lg p-8 text-white">
              {/* Форма платежа */}
              <div className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="block text-sm font-medium text-white">
                        Счет списания *
                      </label>
                      <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={useDebtorFromList}
                          onChange={(e) => {
                            setUseDebtorFromList(e.target.checked);
                            if (!e.target.checked) {
                              setPaymentData({ ...paymentData, debtor_account: "" });
                            }
                          }}
                          className="w-4 h-4"
                        />
                        <span>Выбрать из списка</span>
                      </label>
                    </div>
                    {useDebtorFromList ? (
                      loadingAccounts ? (
                        <div className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 flex items-center gap-2">
                          <Loader2 className="w-4 h-4 animate-spin" />
                          <span>Загрузка счетов...</span>
                        </div>
                      ) : (
                        <select
                          value={paymentData.debtor_account}
                          onChange={(e) => {
                            const selectedAccount = accounts.find(
                              (a) => a.identification === e.target.value
                            );
                            setPaymentData({
                              ...paymentData,
                              debtor_account: selectedAccount?.identification || e.target.value,
                            });
                          }}
                          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          <option value="" className="bg-gray-800 text-white">
                            Выберите счет
                          </option>
                          {accounts
                            .filter((a) => a.bank === selectedBank && a.identification)
                            .map((acc) => (
                              <option
                                key={acc.account_id}
                                value={acc.identification}
                                className="bg-gray-800 text-white"
                              >
                                {acc.nickname || acc.account_id} ({acc.identification})
                              </option>
                            ))}
                        </select>
                      )
                    ) : (
                      <input
                        type="text"
                        value={paymentData.debtor_account}
                        onChange={(e) =>
                          setPaymentData({ ...paymentData, debtor_account: e.target.value })
                        }
                        placeholder="Идентификатор счета списания"
                        className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-white mb-1">
                      Счет получателя *
                    </label>
                    <input
                      type="text"
                      value={paymentData.creditor_account}
                      onChange={(e) =>
                        setPaymentData({ ...paymentData, creditor_account: e.target.value })
                      }
                      placeholder="Идентификатор счета получателя"
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-white mb-1">
                      Сумма *
                    </label>
                    <input
                      type="number"
                      step="0.01"
                      value={paymentData.amount}
                      onChange={(e) =>
                        setPaymentData({ ...paymentData, amount: e.target.value })
                      }
                      placeholder="0.00"
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-white mb-1">
                      Валюта
                    </label>
                    <select
                      value={paymentData.currency}
                      onChange={(e) =>
                        setPaymentData({ ...paymentData, currency: e.target.value })
                      }
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="RUB" className="bg-gray-800 text-white">RUB</option>
                      <option value="USD" className="bg-gray-800 text-white">USD</option>
                      <option value="EUR" className="bg-gray-800 text-white">EUR</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-white mb-1">
                    Назначение платежа
                  </label>
                  <input
                    type="text"
                    value={paymentData.comment}
                    onChange={(e) =>
                      setPaymentData({ ...paymentData, comment: e.target.value })
                    }
                    placeholder="Описание платежа"
                        className="w-full border border-gray-300 rounded-lg px-3 py-2 text-white bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <button
                  onClick={handleCreatePayment}
                  disabled={loading}
                  className="w-full flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition disabled:bg-gray-400"
                >
                  {loading && <Loader2 className="w-5 h-5 animate-spin" />}
                  {loading ? "Создание платежа..." : "Создать платеж"}
                </button>

                {error && (
                  <div className="flex items-center gap-2 text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">
                    <XCircle className="w-5 h-5" />
                    <span>{error}</span>
                  </div>
                )}

                {payment && (
                  <div className="mt-6 p-6 bg-gray-700 rounded-lg border border-gray-600">
                    <h3 className="text-lg font-semibold mb-4 text-white">Информация о платеже</h3>
                    <div className="space-y-2 text-white">
                      <p>
                        <span className="font-medium">ID платежа:</span> {payment.paymentId}
                      </p>
                      <p>
                        <span className="font-medium">Сумма:</span> {payment.amount} {payment.currency}
                      </p>
                      <p>
                        <span className="font-medium">Статус:</span>{" "}
                        <span
                          className={`px-3 py-1 rounded-full text-sm font-medium border ${
                            STATUS_COLORS[payment.status] || "text-gray-300 bg-gray-600 border-gray-500"
                          }`}
                        >
                          {STATUS_RU[payment.status] || payment.status}
                        </span>
                      </p>
                      {payment.creationDateTime && (
                        <p>
                          <span className="font-medium">Дата создания:</span>{" "}
                          {new Date(payment.creationDateTime).toLocaleString("ru-RU")}
                        </p>
                      )}
                      {payment.statusUpdateDateTime && (
                        <p>
                          <span className="font-medium">Последнее обновление:</span>{" "}
                          {new Date(payment.statusUpdateDateTime).toLocaleString("ru-RU")}
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}

      {/* Модальное окно для согласия на перевод */}
      {showConsentModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-lg w-full max-w-md p-6">
            <h2 className="text-2xl font-bold text-blue-700 mb-4">
              Требуется согласие на перевод
            </h2>
            <p className="text-gray-600 mb-6">
              Для создания платежа необходимо согласие на перевод. Данные платежа будут предзаполнены.
            </p>

            <div className="space-y-4 mb-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Счет списания
                </label>
                <input
                  type="text"
                  value={paymentData.debtor_account}
                  disabled
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 bg-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Счет получателя
                </label>
                <input
                  type="text"
                  value={paymentData.creditor_account}
                  disabled
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 bg-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Сумма
                </label>
                <input
                  type="text"
                  value={`${paymentData.amount} ${paymentData.currency}`}
                  disabled
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 bg-gray-100"
                />
              </div>
            </div>

            {consentError && (
              <div className="flex items-center gap-2 text-red-600 bg-red-50 border border-red-200 rounded-lg p-3 mb-4">
                <XCircle className="w-5 h-5" />
                <span>{consentError}</span>
              </div>
            )}

            {consentResult && consentResult.status === "pending" && (
              <div className="flex items-center gap-2 text-yellow-600 bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
                <Loader2 className="w-5 h-5 animate-spin" />
                <div>
                  <p className="font-semibold">Ожидание подтверждения...</p>
                  <p className="text-sm text-gray-600">
                    Подтвердите согласие в приложении банка
                  </p>
                </div>
              </div>
            )}

            {consentResult && consentResult.status === "approved" && (
              <div className="flex items-center gap-2 text-green-600 bg-green-50 border border-green-200 rounded-lg p-4 mb-4">
                <CheckCircle className="w-6 h-6" />
                <div>
                  <p className="font-semibold">Согласие подтверждено</p>
                  <p className="text-sm text-gray-600">
                    Платеж будет создан автоматически
                  </p>
                </div>
              </div>
            )}

            <div className="flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowConsentModal(false);
                  setConsentResult(null);
                  setConsentError(null);
                }}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition"
                disabled={consentLoading || (consentResult && consentResult.status === "pending")}
              >
                Отмена
              </button>
              <button
                onClick={handleCreateConsent}
                disabled={consentLoading || (consentResult && consentResult.status === "pending")}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:bg-gray-400 flex items-center gap-2"
              >
                {consentLoading && <Loader2 className="w-4 h-4 animate-spin" />}
                {consentLoading
                  ? "Создание..."
                  : consentResult && consentResult.status === "pending"
                  ? "Ожидание..."
                  : "Создать согласие"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

