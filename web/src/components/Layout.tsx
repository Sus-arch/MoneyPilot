import { Link, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export default function Layout() {
  const location = useLocation();
  const { token, logout } = useAuth(); // теперь получаем token

  const navItems = [
    { path: "/dashboard", label: "Dashboard" },
    { path: "/accounts", label: "Accounts" },
    { path: "/banks", label: "Banks" },
    { path: "/products", label: "Products" },
  ];

  return (
    <div className="min-h-screen w-full flex flex-col bg-gray-50 text-gray-800">
      <header className="bg-blue-700 text-white py-4 px-6 shadow-md flex justify-between items-center">
        <div className="flex items-center space-x-6">
          <h1 className="text-2xl font-extrabold tracking-wide">
            💰 MoneyPilot
          </h1>
          <nav className="flex space-x-4">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={`relative font-semibold transition-colors duration-200 ${
                  location.pathname === item.path
                    ? "text-yellow-300"
                    : "text-blue-100 hover:text-yellow-200"
                }`}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </div>

        {/* Показываем кнопку только если есть токен */}
        {token && (
          <div className="flex items-center space-x-4">
            <button
              onClick={logout}
              className="px-3 py-1 bg-red-500 rounded hover:bg-red-600 transition"
            >
              Logout
            </button>
          </div>
        )}
      </header>

      <main className="flex-grow p-8">
        <Outlet />
      </main>

      <footer className="bg-blue-700 text-white py-3 text-center text-sm">
        © {new Date().getFullYear()} MoneyPilot
      </footer>
    </div>
  );
}
