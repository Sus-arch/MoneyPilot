import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { post, get } from "../api/client";
import {
  Banknote,
  CreditCard,
  Wallet,
  CheckCircle,
  XCircle,
  Loader2,
  Info,
  X,
  ChevronDown,
  ChevronUp,
} from "lucide-react";

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

const STATUS_RU: Record<string, string> = {
  active: "Активен",
  closed: "Закрыт",
  pending: "В ожидании",
  blocked: "Заблокирован",
};

interface Product {
  agreement_id: string;
  product_id: string;
  product_name: string;
  product_type: string;
}

interface ProductDetails {
  agreement_id: string;
  product_id: string;
  product_name: string;
  product_type: string;
  interest_rate: number;
  amount: number;
  status: string;
}

export default function ProductsPage() {
  const { bankTokens } = useAuth();
  const connectedBanks = BANKS.filter((b) => bankTokens[b.id]);

  const [selectedBank, setSelectedBank] = useState<string | null>(null);
  const [read, setRead] = useState(true);
  const [open, setOpen] = useState(false);
  const [close, setClose] = useState(false);
  const [types, setTypes] = useState<string[]>(["deposit"]);
  const [maxAmount, setMaxAmount] = useState(1000000.0);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<null | any>(null);
  const [error, setError] = useState<string | null>(null);
  const [products, setProducts] = useState<Record<string, Product[]>>({});
  const [loadingProducts, setLoadingProducts] = useState(false);

  const [consentCollapsed, setConsentCollapsed] = useState(false);

  // состояние модального окна
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);
  const [productDetails, setProductDetails] = useState<ProductDetails | null>(null);

  const toggleType = (type: string) => {
    setTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
    );
  };

  const handleSubmit = async (bankId: string) => {
    const token = bankTokens[bankId];
    if (!token) return;

    setError(null);
    setResult(null);

    if (!read && !open && !close) {
      setError("Выберите хотя бы одно действие для согласия.");
      return;
    }

    if (types.length === 0) {
      setError("Выберите хотя бы один тип продукта.");
      return;
    }

    setLoading(true);

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
        "X-Bank-Code": bankId,
      });

      setResult(response);

      if (response.status === "approved") {
        await fetchProducts();
      }
    } catch (err) {
      console.error(err);
      setError("Ошибка при отправке согласия. Попробуйте позже.");
    } finally {
      setLoading(false);
    }
  };

  const fetchProducts = async () => {
    setLoadingProducts(true);
    const all: Record<string, Product[]> = {};

    for (const bank of connectedBanks) {
      const token = bankTokens[bank.id];
      if (!token) continue;

      try {
        const res = await get("/products", {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          "X-Bank-Code": bank.id,
        });

        if (res.data && Array.isArray(res.data)) {
          all[bank.id] = res.data;
        }
      } catch (err) {
        console.error(`Ошибка при получении продуктов ${bank.name}:`, err);
      }
    }

    setProducts(all);
    setLoadingProducts(false);
  };

  const fetchProductDetails = async (bankId: string, agreementId: string) => {
    const token = bankTokens[bankId];
    if (!token) return;

    setModalError(null);
    setModalLoading(true);
    setModalOpen(true);
    setProductDetails(null);

    try {
      const res = await get(`/products/${agreementId}`, {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
        "X-Bank-Code": bankId,
      });

      if (res.data) {
        setProductDetails(res.data);
      } else {
        setModalError("Не удалось загрузить детали продукта.");
      }
    } catch (err) {
      console.error(err);
      setModalError("Ошибка при загрузке деталей продукта.");
    } finally {
      setModalLoading(false);
    }
  };

  useEffect(() => {
    fetchProducts();
  }, [connectedBanks.length]);

  return (
    <div className="max-w-5xl mx-auto mt-10 p-6">
      <h1 className="text-3xl font-bold text-center text-blue-700 mb-8">
        Продукты по подключённым банкам
      </h1>

      {connectedBanks.length === 0 ? (
        <p className="text-center text-gray-600">
          Нет подключённых банков. Подключите банк, чтобы увидеть продукты.
        </p>
      ) : (
        <>
          {/* ===== Выбор банка ===== */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-10">
            {connectedBanks.map((bank) => (
              <div
                key={bank.id}
                onClick={() => setSelectedBank(bank.id)}
                className={`p-6 rounded-2xl border shadow-md flex flex-col items-center justify-between transition transform hover:-translate-y-1 hover:shadow-lg cursor-pointer ${
                  selectedBank === bank.id
                    ? "border-blue-500 bg-blue-50"
                    : "border-gray-300 bg-white"
                }`}
              >
                {bank.icon}
                <h2 className="text-xl font-semibold mt-3">{bank.name}</h2>
                {selectedBank === bank.id && (
                  <CheckCircle className="w-5 h-5 text-blue-600 mt-2" />
                )}
              </div>
            ))}
          </div>

          {/* ===== Настройка согласия (темный стиль + кастомные чекбоксы) ===== */}
          {selectedBank && (
            <div className="bg-gray-800 text-white rounded-2xl shadow-lg mb-10 overflow-hidden">
              <div
                className="flex items-center justify-between cursor-pointer px-6 py-4 bg-gray-900 hover:bg-gray-700 transition"
                onClick={() => setConsentCollapsed(!consentCollapsed)}
              >
                <h2 className="text-2xl font-semibold">
                  Настройка согласия для {selectedBank.toUpperCase()}
                </h2>
                {consentCollapsed ? (
                  <ChevronDown className="w-6 h-6 text-white" />
                ) : (
                  <ChevronUp className="w-6 h-6 text-white" />
                )}
              </div>

              {!consentCollapsed && (
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
                    <label className="block text-sm font-medium text-white mb-1">
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
                    onClick={() => handleSubmit(selectedBank)}
                    disabled={loading}
                    className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-2 rounded-lg font-medium hover:bg-blue-700 transition disabled:bg-gray-400"
                  >
                    {loading && <Loader2 className="w-4 h-4 animate-spin" />}
                    {loading ? "Отправка..." : "Отправить согласие"}
                  </button>

                  {/* Сообщение */}
                  <div className="mt-4 space-y-3">
                    {error && (
                      <div className="flex items-center gap-2 text-red-600 bg-red-50 border border-red-200 rounded-lg p-3">
                        <XCircle className="w-5 h-5" />
                        <span>{error}</span>
                      </div>
                    )}

                    {result && result.status === "approved" && (
                      <div className="flex flex-col sm:flex-row sm:items-center gap-3 text-green-700 bg-green-50 border border-green-200 rounded-lg p-4 shadow-sm">
                        <CheckCircle className="w-6 h-6 text-green-600" />
                        <div>
                          <p className="font-semibold">
                            Согласие подтверждено для{" "}
                            <span className="text-green-800">
                              {result.bank?.toUpperCase()}
                            </span>
                          </p>
                          <p className="text-sm text-gray-600">ID согласия: {result.consent_id}</p>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* ===== Продукты ===== */}
          <div>
            <h2 className="text-2xl font-semibold text-blue-700 mb-6">
              Список продуктов
            </h2>

            {loadingProducts ? (
              <p className="text-center text-gray-500">Загрузка продуктов...</p>
            ) : (
              connectedBanks.map((bank) => (
                <div key={bank.id} className="mb-8">
                  <h3 className="text-xl font-medium text-gray-800 mb-3">
                    {bank.name}
                  </h3>

                  {products[bank.id]?.length ? (
                    <ul className="space-y-3">
                      {products[bank.id].map((p) => (
                        <li
                          key={p.product_id}
                          className="flex items-center justify-between bg-white border border-gray-200 rounded-lg p-4 shadow-sm hover:shadow-md transition"
                        >
                          <div>
                            <p className="font-semibold text-gray-800">
                              {p.product_name}
                            </p>
                            <p className="text-gray-600 text-sm">
                              {PRODUCT_TYPES[p.product_type] || p.product_type}
                            </p>
                          </div>
                          <button
                            onClick={() =>
                              fetchProductDetails(bank.id, p.agreement_id)
                            }
                            className="flex items-center gap-1 text-blue-600 hover:text-blue-800 transition"
                          >
                            <Info className="w-4 h-4" /> Подробнее
                          </button>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-gray-500 text-sm">
                      Нет доступных продуктов.
                    </p>
                  )}
                </div>
              ))
            )}
          </div>
        </>
      )}

      {/* ===== Модальное окно деталей ===== */}
      {modalOpen && (
        <div
          className="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
          onClick={() => setModalOpen(false)}
        >
          <div
            className="bg-white rounded-2xl shadow-lg max-w-md w-full p-6 relative transform transition-all scale-100"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={() => setModalOpen(false)}
              className="absolute top-3 right-3 text-gray-500 hover:text-gray-800"
            >
              <X className="w-5 h-5" />
            </button>

            <h3 className="text-2xl font-semibold text-blue-700 mb-4">
              Детали продукта
            </h3>

            {modalLoading ? (
              <div className="flex items-center justify-center py-10 text-gray-500">
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
                Загрузка данных...
              </div>
            ) : modalError ? (
              <p className="text-red-600">{modalError}</p>
            ) : productDetails ? (
              <div className="space-y-3 text-gray-800">
                <p>
                  <span className="font-semibold">Название:</span>{" "}
                  {productDetails.product_name}
                </p>
                <p>
                  <span className="font-semibold">Тип:</span>{" "}
                  {PRODUCT_TYPES[productDetails.product_type] ||
                    productDetails.product_type}
                </p>
                {productDetails.product_type !== "card" && (
                  <p>
                    <span className="font-semibold">Процентная ставка:</span>{" "}
                    {productDetails.interest_rate}%
                  </p>
                )}
                <p>
                  <span className="font-semibold">Сумма:</span>{" "}
                  {productDetails.amount.toLocaleString("ru-RU")} ₽
                </p>
                <p>
                  <span className="font-semibold">Статус:</span>{" "}
                  {STATUS_RU[productDetails.status] || productDetails.status}
                </p>
              </div>
            ) : (
              <p className="text-gray-500">Нет данных для отображения.</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
