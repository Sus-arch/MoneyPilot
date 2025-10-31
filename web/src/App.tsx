import { Routes, Route, Navigate } from "react-router-dom";
import Layout from "./components/Layout";
import LoginPage from "./pages/LoginPage";
import DashboardPage from "./pages/DashboardPage";
import AccountsPage from "./pages/AccountsPage";
import BanksPage from "./pages/BanksPage";
import ProductsPage from "./pages/ProductsPage";

export default function App() {
  return (
    <Routes>
      {/* Редирект с / на /login */}
      <Route path="/" element={<Navigate to="/login" />} />

      {/* Login без layout */}
      <Route path="/login" element={<LoginPage />} />

      {/* Всё остальное внутри layout */}
      <Route element={<Layout />}>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/accounts" element={<AccountsPage />} />
        <Route path="/banks" element={<BanksPage />} />
        <Route path="/products" element={<ProductsPage />} />
      </Route>
    </Routes>
  );
}
