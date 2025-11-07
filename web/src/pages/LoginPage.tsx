import { useState } from "react";
import { useAuth } from "../context/AuthContext";
import { motion, AnimatePresence } from "framer-motion";

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [bank, setBank] = useState("vbank");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [waiting, setWaiting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    setWaiting(false);

    try {
      const status = await login(email, password, bank);

      if (status === "waiting") {
        setWaiting(true);
      }
    } catch (err) {
      console.error(err);
      setError("Ошибка авторизации. Проверьте данные.");
    } finally {
      if (!waiting) setLoading(false);
    }
  };

  const banks = [
    { code: "vbank", name: "VBank" },
    { code: "abank", name: "ABank" },
    { code: "sbank", name: "SBank" },
  ];

  return (
    <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-blue-800 to-blue-500">
      <div className="bg-white/10 backdrop-blur-lg shadow-lg rounded-2xl p-8 w-full max-w-md border border-white/20">
        <h1 className="text-3xl font-bold text-center text-white mb-6">
          Авторизация в MoneyPilot
        </h1>

        <p className="text-blue-100 text-center mb-6">
          Введите логин, пароль и выберите ваш банк.
        </p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-blue-100">
              Логин
            </label>
            <input
              type="text"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full bg-white/20 text-white border border-blue-300 rounded-lg px-3 py-2 focus:outline-none focus:ring focus:ring-blue-300 placeholder-blue-200"
              placeholder="team200-1"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-blue-100">
              Пароль
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full bg-white/20 text-white border border-blue-300 rounded-lg px-3 py-2 focus:outline-none focus:ring focus:ring-blue-300 placeholder-blue-200"
              placeholder="••••••••"
              required
            />
          </div>

          <div className="relative">
            <label className="block text-sm font-medium text-blue-100 mb-1">
              Банк
            </label>
            <button
              type="button"
              onClick={() => setOpen(!open)}
              className="w-full flex justify-between items-center bg-white/20 text-white border border-blue-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300 hover:bg-white/30 transition-all duration-200"
            >
              {banks.find((b) => b.code === bank)?.name || "Выберите банк"}
              <span className="text-blue-200 ml-2">{open ? "▲" : "▼"}</span>
            </button>

            <AnimatePresence>
              {open && (
                <motion.ul
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  transition={{ duration: 0.2 }}
                  className="absolute z-10 mt-2 w-full bg-white/30 backdrop-blur-xl border border-blue-200 rounded-xl shadow-xl overflow-hidden"
                >
                  {banks.map((b) => (
                    <li
                      key={b.code}
                      onClick={() => {
                        setBank(b.code);
                        setOpen(false);
                      }}
                      className={`px-4 py-2 text-white cursor-pointer hover:bg-blue-600/40 transition ${
                        b.code === bank ? "bg-blue-600/40 font-semibold" : ""
                      }`}
                    >
                      {b.name}
                    </li>
                  ))}
                </motion.ul>
              )}
            </AnimatePresence>
          </div>

          {error && <p className="text-red-400 text-center">{error}</p>}

          {waiting && (
            <p className="text-yellow-300 text-center font-medium animate-pulse">
              ⚠️ Подтвердите согласие в приложении банка, затем вы будете перенаправлены.
            </p>
          )}

          <button
            type="submit"
            disabled={loading || waiting}
            className="w-full bg-blue-600 text-white font-semibold py-2 rounded-lg hover:bg-blue-700 transition disabled:bg-blue-400"
          >
            {waiting
              ? "Ожидание подтверждения..."
              : loading
              ? "Вход..."
              : "Войти"}
          </button>
        </form>
      </div>
    </div>
  );
}
