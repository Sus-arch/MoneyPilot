import { createContext, useState, useEffect, useContext } from "react";
import { post, get } from "../api/client";
import { useNavigate } from "react-router-dom";

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const navigate = useNavigate();
  const [token, setToken] = useState(localStorage.getItem("token") || "");
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (token) loadUser();
    else setLoading(false);
  }, []);

  const login = async (id, password) => {
    const res = await post("/login", { id, password });
    if (!res.token) throw new Error("No token returned from backend");

    setToken(res.token);
    localStorage.setItem("token", res.token);
    await loadUser();
    navigate("/dashboard");
  };

  const logout = () => {
    setToken("");
    setUser(null);
    localStorage.removeItem("token");
    navigate("/login");
  };

  const loadUser = async () => {
    try {
      const profile = await get("/me");
      setUser(profile);
    } catch (err) {
      console.error(err);
      logout();
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{ token, user, loading, login, logout, loadUser }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
