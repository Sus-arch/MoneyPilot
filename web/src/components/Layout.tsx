import { useState } from "react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { Menu, X } from "lucide-react";

export default function Layout() {
  const location = useLocation();
  const { token, logout } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const navItems = [
    { path: "/dashboard", label: "Дашборд" },
    { path: "/banks", label: "Банки" },
    { path: "/accounts", label: "Счета" },
    { path: "/transactions", label: "Транзакции" },
    { path: "/products", label: "Продукты" },
    { path: "/payments", label: "Платежи" },
  ];

  const closeMobileMenu = () => setMobileMenuOpen(false);

  return (
    <div className="min-h-screen w-full flex flex-col bg-gray-50 text-gray-800">
      <header className="bg-blue-700 text-white py-3 px-4 md:py-4 md:px-6 shadow-md">
        <div className="flex justify-between items-center">
          <div className="flex items-center space-x-3 md:space-x-6">
            <h1 className="text-xl md:text-2xl font-extrabold tracking-wide">
              💰 MoneyPilot
            </h1>
            {/* Desktop Navigation */}
            <nav className="hidden lg:flex space-x-4">
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

          <div className="flex items-center space-x-2 md:space-x-4">
            {token && (
              <>
                <button
                  onClick={logout}
                  className="hidden sm:block px-3 py-1 bg-red-500 rounded hover:bg-red-600 transition text-sm md:text-base"
                >
                  Выход
                </button>
                {/* Mobile Menu Button */}
                <button
                  onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                  className="lg:hidden p-2 hover:bg-blue-600 rounded transition"
                  aria-label="Toggle menu"
                >
                  {mobileMenuOpen ? (
                    <X className="w-6 h-6" />
                  ) : (
                    <Menu className="w-6 h-6" />
                  )}
                </button>
              </>
            )}
          </div>
        </div>

        {/* Mobile Navigation */}
        {mobileMenuOpen && (
          <nav className="lg:hidden mt-4 pb-2 border-t border-blue-600 pt-4">
            <div className="flex flex-col space-y-2">
              {navItems.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  onClick={closeMobileMenu}
                  className={`px-4 py-2 rounded font-semibold transition-colors duration-200 ${
                    location.pathname === item.path
                      ? "text-yellow-300 bg-blue-600"
                      : "text-blue-100 hover:text-yellow-200 hover:bg-blue-600"
                  }`}
                >
                  {item.label}
                </Link>
              ))}
              {token && (
                <button
                  onClick={() => {
                    logout();
                    closeMobileMenu();
                  }}
                  className="sm:hidden px-4 py-2 bg-red-500 rounded hover:bg-red-600 transition text-left"
                >
                  Выход
                </button>
              )}
            </div>
          </nav>
        )}
      </header>

      <main className="flex-grow p-4 md:p-6 lg:p-8">
        <Outlet />
      </main>

      <footer className="bg-blue-700 text-white py-2 md:py-3 text-center text-xs md:text-sm">
        © {new Date().getFullYear()} MoneyPilot
      </footer>
    </div>
  );
}
