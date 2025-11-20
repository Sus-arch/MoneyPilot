import { useState, useRef, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import { post, del, get, clearCache } from "../api/client";
import {
  Loader2,
  Banknote,
  CreditCard,
  Wallet,
  CheckCircle,
  XCircle,
  ChevronDown,
  ChevronUp,
  Send,
  FileText,
  X,
} from "lucide-react";

interface Bank {
  id: string;
  name: string;
  connected: boolean;
  status?: "idle" | "pending";
}

interface Account {
  account_id: string;
  nickname: string;
  status: string;
  currency: string;
  bank: string;
  account_subtype: string;
}

const BANKS = [
  { id: "vbank", name: "VBank", icon: <Banknote className="w-8 h-8 text-blue-600" /> },
  { id: "abank", name: "ABank", icon: <CreditCard className="w-8 h-8 text-green-600" /> },
  { id: "sbank", name: "SBank", icon: <Wallet className="w-8 h-8 text-purple-600" /> },
];

const PRODUCT_TYPES: Record<string, string> = {
  card: "Банковская карта",
  credit_card: "Кредитная карта",
  deposit: "Депозит",
  loan: "Кредит",
};

type ConsentTab = "payment" | "product";

type PaymentConsentType = "single_use" | "multi_use" | "vrp";

export default function BanksPage() {
  const { currentBank, bankTokens, saveBankToken } = useAuth();
  const [banks, setBanks] = useState<Bank[]>(
    BANKS.map((b) => ({ ...b, connected: !!bankTokens[b.id], status: "idle" }))
  );
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [showLogin, setShowLogin] = useState<string | null>(null);
  const [credentials, setCredentials] = useState({ email: "", password: "" });
  const wsRef = useRef<WebSocket | null>(null);
  const activeBankRef = useRef<string | null>(null);
  const activeConsentIdRef = useRef<string | null>(null);
  const activeRequestIdRef = useRef<string | null>(null);
  const activeConsentTypeRef = useRef<"payment" | "product" | null>(null);

  // Согласия
  const [selectedBankForConsents, setSelectedBankForConsents] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<ConsentTab>("payment");
  const [consentSectionsCollapsed, setConsentSectionsCollapsed] = useState<Record<string, boolean>>({
    payment: false,
    product: false,
  });

  // Payment Consents
  const [paymentConsentType, setPaymentConsentType] = useState<PaymentConsentType>("single_use");
  const [paymentConsentData, setPaymentConsentData] = useState({
    amount: "",
    currency: "RUB",
    debtor_account: "",
    creditor_account: "",
    creditor_name: "",
    reference: "",
    max_uses: "",
    max_amount_per_payment: "",
    max_total_amount: "",
    allowed_creditor_accounts: [] as string[],
    vrp_max_individual_amount: "",
    vrp_daily_limit: "",
    vrp_monthly_limit: "",
    valid_until: "",
    reason: "",
  });
  const [paymentConsentLoading, setPaymentConsentLoading] = useState(false);
  const [paymentConsentResult, setPaymentConsentResult] = useState<any>(null);
  const [paymentConsentError, setPaymentConsentError] = useState<string | null>(null);
  const [newCreditorAccount, setNewCreditorAccount] = useState("");

  // Product Consents
  const [read, setRead] = useState(true);
  const [open, setOpen] = useState(false);
  const [close, setClose] = useState(false);
  const [types, setTypes] = useState<string[]>(["deposit"]);
  const [maxAmount, setMaxAmount] = useState(1000000.0);
  const [productConsentLoading, setProductConsentLoading] = useState(false);
  const [productConsentResult, setProductConsentResult] = useState<null | any>(null);
  const [productConsentError, setProductConsentError] = useState<string | null>(null);

  // Accounts
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(false);

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
      }));

      setAccounts(accountsData);
    } catch (err: any) {
      console.error("Ошибка загрузки счетов:", err);
    } finally {
      setLoadingAccounts(false);
    }
  };

  useEffect(() => {
    if (selectedBankForConsents && bankTokens[selectedBankForConsents]) {
      fetchAccounts(selectedBankForConsents);
    }
  }, [selectedBankForConsents]);

  // Подключение банка
  const handleConnect = (bankId: string) => setShowLogin(bankId);

  // Отправка логина
  const handleLoginSubmit = async (bankId: string) => {
    setLoading(bankId);
    setMessage("");

    try {
      const response = await post("/auth/login", {
        email: credentials.email,
        password: credentials.password,
        bank: bankId,
      });

      const token = response.token;
      if (!token) throw new Error("JWT токен не получен");

      saveBankToken(bankId, token);

      const consentResponse = await post(
        "/account-consent",
        undefined,
        { "X-Bank-Code": bankId, Authorization: `Bearer ${token}` }
      );

      if (consentResponse.status === "approved") {
        setBanks((prev) =>
          prev.map((b) => (b.id === bankId ? { ...b, connected: true, status: "idle" } : b))
        );
        setMessage(`✅ Банк ${bankId.toUpperCase()} успешно подключён`);
        setLoading(null);
        return;
      }

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, status: "pending" } : b))
      );
      setMessage(`⚠️ Требуется подтверждение в приложении ${bankId.toUpperCase()}`);

      connectWebSocket(
        bankId,
        consentResponse.consent_id || consentResponse.consentId,
        consentResponse.request_id
      );
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit") || err.message?.includes("429")) {
        setMessage("❌ Слишком много запросов. Подождите немного перед повторной попыткой.");
      } else {
        setMessage("❌ Ошибка при подключении банка");
      }
    } finally {
      setLoading(null);
      setShowLogin(null);
    }
  };

  // Отключение банка
  const disconnectBank = async (bankId: string) => {
    const token = bankTokens[bankId];
    if (!token) return;

    setLoading(bankId);
    setMessage("");

    try {
      await del("/account-consent", undefined, {
        "X-Bank-Code": bankId,
        Authorization: `Bearer ${token}`,
      });

      setBanks((prev) =>
        prev.map((b) => (b.id === bankId ? { ...b, connected: false, status: "idle" } : b))
      );

      setMessage(`⚠️ Согласие для ${bankId.toUpperCase()} отозвано`);
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit") || err.message?.includes("429")) {
        setMessage("❌ Слишком много запросов. Подождите немного.");
      } else {
        setMessage("❌ Ошибка при отзыве согласия");
      }
    } finally {
      setLoading(null);
    }
  };

  // Подключение WebSocket
  const connectWebSocket = (bankId: string, consentId?: string, requestId?: string, consentType?: "payment" | "product") => {
    if (wsRef.current) wsRef.current.close();

    const socket = new WebSocket("ws://localhost:8080/ws");
    wsRef.current = socket;
    activeBankRef.current = bankId;
    if (consentId) activeConsentIdRef.current = consentId;
    if (requestId) activeRequestIdRef.current = requestId;
    if (consentType) activeConsentTypeRef.current = consentType;
    
    console.log(`🔌 WebSocket подключен для ${bankId}, consent_id: ${consentId || 'нет'}, request_id: ${requestId || 'нет'}, type: ${consentType || 'account'}`);

    socket.onopen = () => console.log(`✅ WebSocket для ${bankId} открыт`);
    socket.onclose = () => {
      console.log(`❌ WebSocket закрыт для ${bankId}`);
      wsRef.current = null;
      activeBankRef.current = null;
      activeConsentIdRef.current = null;
      activeRequestIdRef.current = null;
      activeConsentTypeRef.current = null;
    };

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        console.log("📨 WebSocket message:", msg);

        if (!msg.consent_id || !msg.status) return;

        const currentBank = activeBankRef.current;
        if (!currentBank) return;

        // Проверяем, относится ли сообщение к нашему согласию
        // Poller может отправить обновленный consent_id, поэтому проверяем оба
        // Если есть активное согласие (payment или product), проверяем ID
        // Если нет активного согласия, принимаем все сообщения для текущего банка (для account consents)
        if (activeConsentIdRef.current || activeRequestIdRef.current) {
          const matchesConsentId = activeConsentIdRef.current && msg.consent_id === activeConsentIdRef.current;
          const matchesRequestId = activeRequestIdRef.current && msg.consent_id === activeRequestIdRef.current;
          
          console.log(`🔍 Проверка WebSocket сообщения: consent_id=${msg.consent_id}, активный consent_id=${activeConsentIdRef.current || 'нет'}, активный request_id=${activeRequestIdRef.current || 'нет'}, type=${activeConsentTypeRef.current || 'account'}, matchesConsentId=${matchesConsentId}, matchesRequestId=${matchesRequestId}`);
          
          // Для payment и product согласий: если есть активное согласие этого типа для текущего банка,
          // и приходит approved статус, принимаем его даже если ID не совпадают точно
          // (poller может отправить обновленный consent_id, который был создан из request_id)
          const isPaymentOrProduct = activeConsentTypeRef.current === "payment" || activeConsentTypeRef.current === "product";
          const isApprovedStatus = msg.status === "approved";
          
          if (!matchesConsentId && !matchesRequestId) {
            // Для account consents: если activeConsentTypeRef не установлен (null), это account consent
            const isAccountConsent = activeConsentTypeRef.current === null;
            // Для payment/product согласий: если есть активное согласие этого типа и статус approved
            if ((isPaymentOrProduct && isApprovedStatus) || (isAccountConsent && isApprovedStatus)) {
              console.log(`✅ Принимаем обновленный consent_id для ${activeConsentTypeRef.current || 'account'} согласия: ${msg.consent_id} (был: ${activeConsentIdRef.current || activeRequestIdRef.current})`);
              // Продолжаем обработку
            } else {
              console.log(`⚠️ WebSocket message не соответствует активному согласию. Ожидали: ${activeConsentIdRef.current || activeRequestIdRef.current}, получили: ${msg.consent_id}`);
              return;
            }
          }
        } else {
          console.log(`ℹ️ Нет активного согласия, принимаем сообщение для банка ${currentBank}`);
        }

        if (msg.status === "pending") {
          setBanks((prev) =>
            prev.map((b) => (b.id === currentBank ? { ...b, status: "pending" } : b))
          );
          setMessage(`⚠️ Подтверждение для ${currentBank.toUpperCase()} ожидается`);
          
          // Обновляем результат для payment/product согласий
          if (activeConsentTypeRef.current === "payment" || activeTab === "payment") {
            setPaymentConsentResult({ ...msg, status: "pending" });
          } else if (activeConsentTypeRef.current === "product" || activeTab === "product") {
            setProductConsentResult({ ...msg, status: "pending" });
          }
        }

        if (msg.status === "approved") {
          setBanks((prev) =>
            prev.map((b) =>
              b.id === currentBank ? { ...b, connected: true, status: "idle" } : b
            )
          );
          setMessage(`✅ Согласие для ${currentBank.toUpperCase()} подтверждено`);
          
          // Обновляем результат в зависимости от типа согласия
          if (activeConsentTypeRef.current === "payment" || activeTab === "payment") {
            setPaymentConsentResult({ ...msg, status: "approved" });
            setPaymentConsentError(null);
          } else if (activeConsentTypeRef.current === "product" || activeTab === "product") {
            setProductConsentResult({ ...msg, status: "approved" });
            setProductConsentError(null);
          }
          
          socket.close();
        }
      } catch (err) {
        console.error("Ошибка при обработке WebSocket-сообщения:", err);
      }
    };
  };

  // Создание Payment Consent
  const handlePaymentConsentSubmit = async () => {
    if (!selectedBankForConsents) {
      setPaymentConsentError("Выберите банк");
      return;
    }

    const token = bankTokens[selectedBankForConsents];
    if (!token) {
      setPaymentConsentError("Банк не подключен");
      return;
    }

    if (!paymentConsentData.debtor_account) {
      setPaymentConsentError("Выберите счет списания");
      return;
    }

    setPaymentConsentError(null);
    setPaymentConsentResult(null);
    setPaymentConsentLoading(true);

    try {
      const body: any = {
        consent_type: paymentConsentType,
        debtor_account: paymentConsentData.debtor_account,
        currency: paymentConsentData.currency,
      };

      if (paymentConsentType === "single_use") {
        if (!paymentConsentData.amount) {
          setPaymentConsentError("Укажите сумму");
          setPaymentConsentLoading(false);
          return;
        }
        body.amount = parseFloat(paymentConsentData.amount);
        if (paymentConsentData.creditor_account) {
          body.creditor_account = paymentConsentData.creditor_account;
        }
        if (paymentConsentData.creditor_name) {
          body.creditor_name = paymentConsentData.creditor_name;
        }
        if (paymentConsentData.reference) {
          body.reference = paymentConsentData.reference;
        }
      } else if (paymentConsentType === "multi_use") {
        if (!paymentConsentData.max_uses || !paymentConsentData.max_amount_per_payment || !paymentConsentData.max_total_amount) {
          setPaymentConsentError("Заполните все обязательные поля для multi_use");
          setPaymentConsentLoading(false);
          return;
        }
        body.max_uses = parseInt(paymentConsentData.max_uses);
        body.max_amount_per_payment = parseFloat(paymentConsentData.max_amount_per_payment);
        body.max_total_amount = parseFloat(paymentConsentData.max_total_amount);
        if (paymentConsentData.allowed_creditor_accounts.length > 0) {
          body.allowed_creditor_accounts = paymentConsentData.allowed_creditor_accounts;
        }
        if (paymentConsentData.valid_until) {
          // Преобразуем datetime-local в ISO 8601 формат
          const date = new Date(paymentConsentData.valid_until);
          body.valid_until = date.toISOString();
        }
      } else if (paymentConsentType === "vrp") {
        if (!paymentConsentData.vrp_max_individual_amount || !paymentConsentData.vrp_daily_limit || !paymentConsentData.vrp_monthly_limit) {
          setPaymentConsentError("Заполните все обязательные поля для vrp");
          setPaymentConsentLoading(false);
          return;
        }
        body.vrp_max_individual_amount = parseFloat(paymentConsentData.vrp_max_individual_amount);
        body.vrp_daily_limit = parseFloat(paymentConsentData.vrp_daily_limit);
        body.vrp_monthly_limit = parseFloat(paymentConsentData.vrp_monthly_limit);
        if (paymentConsentData.valid_until) {
          // Преобразуем datetime-local в ISO 8601 формат
          const date = new Date(paymentConsentData.valid_until);
          body.valid_until = date.toISOString();
        }
      }

      if (paymentConsentData.reason) {
        body.reason = paymentConsentData.reason;
      }

      const response = await post("/payment-consents/request", body, {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": selectedBankForConsents,
      });

      setPaymentConsentResult(response);

      if (response.status === "approved") {
        setMessage(`✅ Согласие на переводы для ${selectedBankForConsents.toUpperCase()} подтверждено`);
      } else if (!response.auto_approved) {
        // Если не автоподтверждено, ждем через WebSocket
        // Сохраняем оба ID для проверки, так как poller может отправить обновленный consent_id
        connectWebSocket(
          selectedBankForConsents,
          response.consent_id,
          response.request_id,
          "payment"
        );
        setMessage(`⚠️ Требуется подтверждение согласия на переводы в приложении ${selectedBankForConsents.toUpperCase()}`);
      }
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setPaymentConsentError("Слишком много запросов. Подождите немного.");
      } else {
        setPaymentConsentError("Ошибка при отправке согласия. Попробуйте позже.");
      }
    } finally {
      setPaymentConsentLoading(false);
    }
  };

  // Создание Product Consent
  const handleProductConsentSubmit = async () => {
    if (!selectedBankForConsents) {
      setProductConsentError("Выберите банк");
      return;
    }

    const token = bankTokens[selectedBankForConsents];
    if (!token) {
      setProductConsentError("Банк не подключен");
      return;
    }

    setProductConsentError(null);
    setProductConsentResult(null);

    if (!read && !open && !close) {
      setProductConsentError("Выберите хотя бы одно действие для согласия.");
      return;
    }

    if (types.length === 0) {
      setProductConsentError("Выберите хотя бы один тип продукта.");
      return;
    }

    setProductConsentLoading(true);

    try {
      const body = {
        read_product_agreements: read,
        open_product_agreements: open,
        close_product_agreements: close,
        allowed_product_types: types,
        max_amount: maxAmount,
      };

      const response = await post("/product-consents/request", body, {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": selectedBankForConsents,
      });

      setProductConsentResult(response);

      if (response.status === "approved") {
        clearCache("/products");
        setMessage(`✅ Согласие на продукты для ${selectedBankForConsents.toUpperCase()} подтверждено`);
      } else if (!response.auto_approved) {
        // Сохраняем оба ID для проверки, так как poller может отправить обновленный consent_id
        // Для product consents poller отправляет request_id, который потом обновляется на consent_id
        connectWebSocket(
          selectedBankForConsents,
          response.consent_id,
          response.request_id,
          "product"
        );
        setMessage(`⚠️ Требуется подтверждение согласия на продукты в приложении ${selectedBankForConsents.toUpperCase()}`);
      }
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("Rate limit")) {
        setProductConsentError("Слишком много запросов. Подождите немного.");
      } else {
        setProductConsentError("Ошибка при отправке согласия. Попробуйте позже.");
      }
    } finally {
      setProductConsentLoading(false);
    }
  };

  const toggleType = (type: string) => {
    setTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
    );
  };

  const addCreditorAccount = () => {
    if (newCreditorAccount.trim()) {
      setPaymentConsentData((prev) => ({
        ...prev,
        allowed_creditor_accounts: [...prev.allowed_creditor_accounts, newCreditorAccount.trim()],
      }));
      setNewCreditorAccount("");
    }
  };

  const removeCreditorAccount = (index: number) => {
    setPaymentConsentData((prev) => ({
      ...prev,
      allowed_creditor_accounts: prev.allowed_creditor_accounts.filter((_, i) => i !== index),
    }));
  };

  const connectedBanks = banks.filter((b) => b.connected);

  return (
    <div className="max-w-6xl mx-auto mt-10 p-6">
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-8">
        Управление банками и согласиями
      </h1>

      {/* Банки */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-10">
        {banks.map((bank) => {
          const Icon = BANKS.find((b) => b.id === bank.id)?.icon;
          const isPending = bank.status === "pending";
          return (
            <div
              key={bank.id}
              className={`p-6 rounded-2xl border shadow-md flex flex-col items-center justify-between transition transform hover:-translate-y-1 hover:shadow-lg relative ${
                bank.connected
                  ? "border-green-500 bg-green-50"
                  : isPending
                  ? "border-yellow-400 bg-yellow-50 animate-pulse"
                  : "border-gray-300 bg-gray-50"
              }`}
            >
              {isPending && (
                <div className="absolute top-4 right-4 animate-spin">
                  <Loader2 className="w-5 h-5 text-yellow-500" />
                </div>
              )}
              {Icon}
              <h2 className="text-xl font-semibold mt-3">{bank.name}</h2>

              {bank.id === currentBank && (
                <span className="text-blue-600 font-medium text-sm mt-1">
                  (основной)
                </span>
              )}

              {bank.connected ? (
                <button
                  onClick={() => disconnectBank(bank.id)}
                  disabled={!!loading || isPending}
                  className="mt-4 px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition disabled:bg-gray-400"
                >
                  {loading === bank.id ? "Отключение..." : "Отключить"}
                </button>
              ) : (
                <button
                  onClick={() => handleConnect(bank.id)}
                  disabled={!!loading || isPending}
                  className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:bg-gray-400"
                >
                  {loading === bank.id ? "Подключение..." : "Подключить"}
                </button>
              )}
            </div>
          );
        })}
      </div>

      {/* Модальное окно логина */}
      {showLogin && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-gray-800 p-6 rounded-xl shadow-lg w-96 text-white">
            <h2 className="text-xl font-bold mb-4">
              Вход в {showLogin.toUpperCase()}
            </h2>

            <input
              type="text"
              placeholder="Логин"
              value={credentials.email}
              onChange={(e) =>
                setCredentials({ ...credentials, email: e.target.value })
              }
              className="border border-gray-600 w-full mb-3 px-3 py-2 rounded bg-gray-700 text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            <input
              type="password"
              placeholder="Пароль"
              value={credentials.password}
              onChange={(e) =>
                setCredentials({ ...credentials, password: e.target.value })
              }
              className="border border-gray-600 w-full mb-3 px-3 py-2 rounded bg-gray-700 text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowLogin(null)}
                className="px-3 py-1 rounded bg-gray-500 hover:bg-gray-600 transition"
              >
                Отмена
              </button>
              <button
                onClick={() => handleLoginSubmit(showLogin)}
                className="px-3 py-1 rounded bg-blue-600 text-white hover:bg-blue-700 transition"
              >
                Подключить
              </button>
            </div>
          </div>
        </div>
      )}

      {message && (
        <p className="text-center mt-6 text-gray-700 font-medium mb-6">{message}</p>
      )}

      {/* Согласия */}
      {connectedBanks.length > 0 && (
        <div className="mt-10">
          <h2 className="text-2xl font-bold text-center text-blue-700 mb-6">
            Управление согласиями
          </h2>

          {/* Выбор банка для согласий */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Выберите банк для работы с согласиями
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              {connectedBanks.map((bank) => (
                <button
                  key={bank.id}
                  onClick={() => {
                    setSelectedBankForConsents(bank.id);
                    fetchAccounts(bank.id);
                  }}
                  className={`p-4 rounded-lg border-2 transition ${
                    selectedBankForConsents === bank.id
                      ? "border-blue-500 bg-blue-50"
                      : "border-gray-300 bg-white hover:border-gray-400"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {BANKS.find((b) => b.id === bank.id)?.icon}
                    <span className="font-medium">{bank.name}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {selectedBankForConsents && (
            <>
              {/* Табы */}
              <div className="flex gap-2 mb-6 border-b">
                <button
                  onClick={() => setActiveTab("payment")}
                  className={`px-6 py-3 font-medium transition ${
                    activeTab === "payment"
                      ? "border-b-2 border-blue-500 text-blue-600"
                      : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  <Send className="w-5 h-5 inline mr-2" />
                  Согласия на переводы
                </button>
                <button
                  onClick={() => setActiveTab("product")}
                  className={`px-6 py-3 font-medium transition ${
                    activeTab === "product"
                      ? "border-b-2 border-blue-500 text-blue-600"
                      : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  <FileText className="w-5 h-5 inline mr-2" />
                  Согласия на продукты
                </button>
              </div>

              {/* Payment Consents */}
              {activeTab === "payment" && (
                <div className="bg-gray-800 text-white rounded-2xl shadow-lg overflow-hidden">
                  <div
                    className="flex items-center justify-between cursor-pointer px-6 py-4 bg-gray-900 hover:bg-gray-700 transition"
                    onClick={() =>
                      setConsentSectionsCollapsed({
                        ...consentSectionsCollapsed,
                        payment: !consentSectionsCollapsed.payment,
                      })
                    }
                  >
                    <h3 className="text-xl font-semibold">
                      Создание согласия на переводы для {selectedBankForConsents.toUpperCase()}
                    </h3>
                    {consentSectionsCollapsed.payment ? (
                      <ChevronDown className="w-6 h-6" />
                    ) : (
                      <ChevronUp className="w-6 h-6" />
                    )}
                  </div>

                  {!consentSectionsCollapsed.payment && (
                    <div className="p-8 space-y-6">
                      {/* Тип согласия */}
                      <div>
                        <label className="block text-sm font-medium mb-2">Тип согласия</label>
                        <div className="flex gap-4">
                          {(["single_use", "multi_use", "vrp"] as PaymentConsentType[]).map(
                            (type) => (
                              <label
                                key={type}
                                className="flex items-center gap-2 cursor-pointer"
                              >
                                <input
                                  type="radio"
                                  name="consent_type"
                                  value={type}
                                  checked={paymentConsentType === type}
                                  onChange={() => setPaymentConsentType(type)}
                                  className="accent-blue-500"
                                />
                                <span>
                                  {type === "single_use"
                                    ? "Одноразовый"
                                    : type === "multi_use"
                                    ? "Многоразовый"
                                    : "VRP (переменные платежи)"}
                                </span>
                              </label>
                            )
                          )}
                        </div>
                      </div>

                      {/* Общие поля */}
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium mb-1">
                            Счет списания (обязательно) *
                          </label>
                          {loadingAccounts ? (
                            <Loader2 className="w-5 h-5 animate-spin" />
                          ) : (
                            <select
                              value={paymentConsentData.debtor_account}
                              onChange={(e) =>
                                setPaymentConsentData({
                                  ...paymentConsentData,
                                  debtor_account: e.target.value,
                                })
                              }
                              className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                            >
                              <option value="">Выберите счет</option>
                              {accounts
                                .filter((a) => a.bank === selectedBankForConsents)
                                .map((acc) => (
                                  <option key={acc.account_id} value={acc.account_id}>
                                    {acc.nickname || acc.account_id} ({acc.account_subtype})
                                  </option>
                                ))}
                            </select>
                          )}
                        </div>

                        <div>
                          <label className="block text-sm font-medium mb-1">Валюта</label>
                          <select
                            value={paymentConsentData.currency}
                            onChange={(e) =>
                              setPaymentConsentData({
                                ...paymentConsentData,
                                currency: e.target.value,
                              })
                            }
                            className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                          >
                            <option value="RUB">RUB</option>
                            <option value="USD">USD</option>
                            <option value="EUR">EUR</option>
                          </select>
                        </div>
                      </div>

                      {/* Single Use */}
                      {paymentConsentType === "single_use" && (
                        <>
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Сумма (обязательно) *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.amount}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    amount: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Счет получателя (опционально)
                              </label>
                              <select
                                value={paymentConsentData.creditor_account}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    creditor_account: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              >
                                <option value="">Любой счет</option>
                                {accounts
                                  .filter((a) => a.bank === selectedBankForConsents)
                                  .map((acc) => (
                                    <option key={acc.account_id} value={acc.account_id}>
                                      {acc.nickname || acc.account_id}
                                    </option>
                                  ))}
                              </select>
                            </div>
                          </div>
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Имя получателя
                              </label>
                              <input
                                type="text"
                                value={paymentConsentData.creditor_name}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    creditor_name: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">Назначение</label>
                              <input
                                type="text"
                                value={paymentConsentData.reference}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    reference: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                          </div>
                        </>
                      )}

                      {/* Multi Use */}
                      {paymentConsentType === "multi_use" && (
                        <>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Макс. использований *
                              </label>
                              <input
                                type="number"
                                value={paymentConsentData.max_uses}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    max_uses: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Макс. сумма за платеж *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.max_amount_per_payment}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    max_amount_per_payment: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Макс. общая сумма *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.max_total_amount}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    max_total_amount: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                          </div>
                          <div>
                            <label className="block text-sm font-medium mb-1">
                              Разрешенные счета получателей (опционально)
                            </label>
                            <div className="flex gap-2 mb-2">
                              <input
                                type="text"
                                value={newCreditorAccount}
                                onChange={(e) => setNewCreditorAccount(e.target.value)}
                                placeholder="Номер счета"
                                className="flex-1 border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                onKeyPress={(e) => e.key === "Enter" && addCreditorAccount()}
                              />
                              <button
                                onClick={addCreditorAccount}
                                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                              >
                                Добавить
                              </button>
                            </div>
                            {paymentConsentData.allowed_creditor_accounts.length > 0 && (
                              <div className="flex flex-wrap gap-2">
                                {paymentConsentData.allowed_creditor_accounts.map((acc, idx) => (
                                  <span
                                    key={idx}
                                    className="flex items-center gap-2 bg-gray-700 px-3 py-1 rounded-lg"
                                  >
                                    {acc}
                                    <button
                                      onClick={() => removeCreditorAccount(idx)}
                                      className="text-red-400 hover:text-red-300"
                                    >
                                      <X className="w-4 h-4" />
                                    </button>
                                  </span>
                                ))}
                              </div>
                            )}
                          </div>
                          <div>
                            <label className="block text-sm font-medium mb-1">
                              Действительно до
                            </label>
                            <input
                              type="datetime-local"
                              value={paymentConsentData.valid_until}
                              onChange={(e) =>
                                setPaymentConsentData({
                                  ...paymentConsentData,
                                  valid_until: e.target.value,
                                })
                              }
                              className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                            />
                          </div>
                        </>
                      )}

                      {/* VRP */}
                      {paymentConsentType === "vrp" && (
                        <>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Макс. сумма за платеж *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.vrp_max_individual_amount}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    vrp_max_individual_amount: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Дневной лимит *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.vrp_daily_limit}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    vrp_daily_limit: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium mb-1">
                                Месячный лимит *
                              </label>
                              <input
                                type="number"
                                step="0.01"
                                value={paymentConsentData.vrp_monthly_limit}
                                onChange={(e) =>
                                  setPaymentConsentData({
                                    ...paymentConsentData,
                                    vrp_monthly_limit: e.target.value,
                                  })
                                }
                                className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                          </div>
                          <div>
                            <label className="block text-sm font-medium mb-1">
                              Действительно до
                            </label>
                            <input
                              type="datetime-local"
                              value={paymentConsentData.valid_until}
                              onChange={(e) =>
                                setPaymentConsentData({
                                  ...paymentConsentData,
                                  valid_until: e.target.value,
                                })
                              }
                              className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                            />
                          </div>
                        </>
                      )}

                      {/* Причина (опционально для всех типов) */}
                      <div>
                        <label className="block text-sm font-medium mb-1">Причина (опционально)</label>
                        <input
                          type="text"
                          value={paymentConsentData.reason}
                          onChange={(e) =>
                            setPaymentConsentData({
                              ...paymentConsentData,
                              reason: e.target.value,
                            })
                          }
                          className="w-full border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>

                      <button
                        onClick={handlePaymentConsentSubmit}
                        disabled={paymentConsentLoading}
                        className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-2 rounded-lg font-medium hover:bg-blue-700 transition disabled:bg-gray-400 w-full"
                      >
                        {paymentConsentLoading && <Loader2 className="w-4 h-4 animate-spin" />}
                        {paymentConsentLoading ? "Отправка..." : "Отправить согласие"}
                      </button>

                      {/* Результаты */}
                      {paymentConsentError && (
                        <div className="flex items-center gap-2 text-red-400 bg-red-900/30 border border-red-500 rounded-lg p-3">
                          <XCircle className="w-5 h-5" />
                          <span>{paymentConsentError}</span>
                        </div>
                      )}

                      {paymentConsentResult && paymentConsentResult.status === "approved" && (
                        <div className="flex items-center gap-2 text-green-400 bg-green-900/30 border border-green-500 rounded-lg p-4">
                          <CheckCircle className="w-6 h-6" />
                          <div>
                            <p className="font-semibold">
                              Согласие подтверждено для{" "}
                              {paymentConsentResult.bank?.toUpperCase()}
                            </p>
                            <p className="text-sm text-gray-300">
                              ID согласия: {paymentConsentResult.consent_id || paymentConsentResult.request_id}
                            </p>
                          </div>
                        </div>
                      )}

                      {paymentConsentResult && paymentConsentResult.status === "pending" && (
                        <div className="flex items-center gap-2 text-yellow-400 bg-yellow-900/30 border border-yellow-500 rounded-lg p-4">
                          <Loader2 className="w-5 h-5 animate-spin" />
                          <div>
                            <p className="font-semibold">Ожидание подтверждения...</p>
                            <p className="text-sm text-gray-300">
                              ID запроса: {paymentConsentResult.request_id}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* Product Consents */}
              {activeTab === "product" && (
                <div className="bg-gray-800 text-white rounded-2xl shadow-lg overflow-hidden">
                  <div
                    className="flex items-center justify-between cursor-pointer px-6 py-4 bg-gray-900 hover:bg-gray-700 transition"
                    onClick={() =>
                      setConsentSectionsCollapsed({
                        ...consentSectionsCollapsed,
                        product: !consentSectionsCollapsed.product,
                      })
                    }
                  >
                    <h3 className="text-xl font-semibold">
                      Настройка согласия на продукты для {selectedBankForConsents.toUpperCase()}
                    </h3>
                    {consentSectionsCollapsed.product ? (
                      <ChevronDown className="w-6 h-6" />
                    ) : (
                      <ChevronUp className="w-6 h-6" />
                    )}
                  </div>

                  {!consentSectionsCollapsed.product && (
                    <div className="p-8 space-y-6">
                      {/* Тип согласий */}
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        {[
                          { label: "Чтение продуктов", value: read, setValue: setRead },
                          { label: "Открытие продуктов", value: open, setValue: setOpen },
                          { label: "Закрытие продуктов", value: close, setValue: setClose },
                        ].map((item) => (
                          <label
                            key={item.label}
                            className="flex items-center gap-2 cursor-pointer select-none"
                          >
                            <input
                              type="checkbox"
                              checked={item.value}
                              onChange={() => item.setValue(!item.value)}
                              className="accent-blue-500 w-5 h-5 rounded transition"
                            />
                            <span>{item.label}</span>
                          </label>
                        ))}
                      </div>

                      {/* Типы продуктов */}
                      <div>
                        <h3 className="text-lg font-medium mb-2">Типы продуктов</h3>
                        <div className="flex gap-4 flex-wrap">
                          {Object.entries(PRODUCT_TYPES).map(([key, label]) => (
                            <label
                              key={key}
                              className="flex items-center gap-2 cursor-pointer select-none"
                            >
                              <input
                                type="checkbox"
                                checked={types.includes(key)}
                                onChange={() => toggleType(key)}
                                className="accent-blue-500 w-5 h-5 rounded transition"
                              />
                              <span>{label}</span>
                            </label>
                          ))}
                        </div>
                      </div>

                      {/* Сумма */}
                      <div>
                        <label className="block text-sm font-medium mb-1">
                          Максимальная сумма продукта (₽)
                        </label>
                        <input
                          type="number"
                          value={maxAmount}
                          onChange={(e) => setMaxAmount(parseFloat(e.target.value))}
                          min={1}
                          step={1000}
                          className="w-60 border border-gray-600 rounded-lg px-3 py-2 bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>

                      <button
                        onClick={handleProductConsentSubmit}
                        disabled={productConsentLoading}
                        className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-2 rounded-lg font-medium hover:bg-blue-700 transition disabled:bg-gray-400"
                      >
                        {productConsentLoading && <Loader2 className="w-4 h-4 animate-spin" />}
                        {productConsentLoading ? "Отправка..." : "Отправить согласие"}
                      </button>

                      {/* Результаты */}
                      {productConsentError && (
                        <div className="flex items-center gap-2 text-red-400 bg-red-900/30 border border-red-500 rounded-lg p-3">
                          <XCircle className="w-5 h-5" />
                          <span>{productConsentError}</span>
                        </div>
                      )}

                      {productConsentResult && productConsentResult.status === "approved" && (
                        <div className="flex items-center gap-2 text-green-400 bg-green-900/30 border border-green-500 rounded-lg p-4">
                          <CheckCircle className="w-6 h-6" />
                          <div>
                            <p className="font-semibold">
                              Согласие подтверждено для{" "}
                              {productConsentResult.bank?.toUpperCase()}
                            </p>
                            <p className="text-sm text-gray-300">
                              ID согласия: {productConsentResult.consent_id}
                            </p>
                          </div>
                        </div>
                      )}

                      {productConsentResult && productConsentResult.status === "pending" && (
                        <div className="flex items-center gap-2 text-yellow-400 bg-yellow-900/30 border border-yellow-500 rounded-lg p-4">
                          <Loader2 className="w-5 h-5 animate-spin" />
                          <div>
                            <p className="font-semibold">Ожидание подтверждения...</p>
                            <p className="text-sm text-gray-300">
                              ID запроса: {productConsentResult.consent_id}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
