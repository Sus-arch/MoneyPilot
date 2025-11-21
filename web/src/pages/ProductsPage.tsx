import { useEffect, useState, useRef } from "react";
import { useAuth } from "../context/AuthContext";
import { get } from "../api/client";
import {
  Banknote,
  CreditCard,
  Wallet,
  Loader2,
  Info,
  X,
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

  const [products, setProducts] = useState<Record<string, Product[]>>({});
  const [loadingProducts, setLoadingProducts] = useState(false);
  const fetchingProductsRef = useRef(false);
  const lastBanksRef = useRef<string>("");

  // состояние модального окна
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);
  const [productDetails, setProductDetails] = useState<ProductDetails | null>(null);

  const fetchProducts = async (force = false) => {
    // Предотвращаем дублирующие запросы
    if (fetchingProductsRef.current) return;
    
    const banksKey = connectedBanks.map(b => b.id).sort().join(",");
    // Если не принудительное обновление и ключ банков не изменился, пропускаем
    if (!force && lastBanksRef.current === banksKey && Object.keys(products).length > 0) {
      return;
    }
    
    fetchingProductsRef.current = true;
    setLoadingProducts(true);
    const all: Record<string, Product[]> = {};

    // Обрабатываем банки последовательно, чтобы не перегружать API
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
      } catch (err: any) {
        console.error(`Ошибка при получении продуктов ${bank.name}:`, err);
        if (err.message?.includes("Rate limit")) {
          // Пропускаем этот банк при rate limit
          continue;
        }
      }
    }

    setProducts(all);
    lastBanksRef.current = banksKey;
    setLoadingProducts(false);
    fetchingProductsRef.current = false;
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
    // Обновляем продукты при изменении списка подключенных банков
    const banksKey = connectedBanks.map(b => b.id).sort().join(",");
    if (lastBanksRef.current !== banksKey) {
      fetchProducts(false);
    }
  }, [connectedBanks.length, bankTokens]);

  return (
    <div className="max-w-5xl mx-auto mt-4 md:mt-10 p-4 md:p-6">
      <h1 className="text-2xl md:text-3xl font-bold text-center text-blue-700 mb-6 md:mb-8">
        Продукты по подключённым банкам
      </h1>

      {connectedBanks.length === 0 ? (
        <p className="text-center text-gray-600">
          Нет подключённых банков. Подключите банк, чтобы увидеть продукты.
        </p>
      ) : (
        <>
          {/* ===== Продукты ===== */}
          <div>
            <h2 className="text-xl md:text-2xl font-semibold text-blue-700 mb-4 md:mb-6">
              Список продуктов
            </h2>

            {loadingProducts ? (
              <p className="text-center text-gray-500">Загрузка продуктов...</p>
            ) : (
              connectedBanks.map((bank) => (
                <div key={bank.id} className="mb-6 md:mb-8">
                  <h3 className="text-lg md:text-xl font-medium text-gray-800 mb-3">
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
            className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
            onClick={() => setModalOpen(false)}
          >
            <div
              className="bg-white rounded-2xl shadow-lg max-w-md w-full p-4 md:p-6 relative transform transition-all scale-100 max-h-[90vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
            <button
              onClick={() => setModalOpen(false)}
              className="absolute top-3 right-3 text-gray-500 hover:text-gray-800"
            >
              <X className="w-5 h-5" />
            </button>

            <h3 className="text-xl md:text-2xl font-semibold text-blue-700 mb-4">
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
