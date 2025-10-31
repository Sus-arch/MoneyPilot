import { Link, Outlet, useLocation } from "react-router-dom";

export default function Layout() {
  const location = useLocation();

  const navItems = [
    { path: "/dashboard", label: "Дэшборд" },
    { path: "/accounts", label: "Счета" },
    { path: "/banks", label: "Банки" },
    { path: "/products", label: "Продукты" },
  ];

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 text-gray-800">
      {/* Header */}
      <header className="bg-blue-600 text-white py-4 px-6 shadow-md flex justify-between items-center">
        <h1 className="text-2xl font-bold">💰 MoneyPilot</h1>
        <nav className="space-x-6">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`relative font-semibold transition-colors duration-200 ${
                location.pathname === item.path
                  ? "text-yellow-300 after:absolute after:bottom-[-4px] after:left-0 after:w-full after:h-[2px] after:bg-yellow-300"
                  : "text-blue-100 hover:text-yellow-200"
              }`}
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </header>

      {/* Main content */}
      <main className="flex-grow p-8">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="bg-blue-600 text-white py-3 text-center">
        © {new Date().getFullYear()} MoneyPilot
      </footer>
    </div>
  );
}
