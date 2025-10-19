// contexts/AuthContext.tsx
import { createContext, useContext, useState, useEffect, ReactNode } from "react";

interface AuthContextType {
  isSignedIn: boolean;
  loading: boolean;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isSignedIn, setIsSignedIn] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check authentication status on mount
    const token = localStorage.getItem("token");
    setIsSignedIn(!!token);
    setLoading(false);
  }, []);

  const login = (token: string) => {
    localStorage.setItem("token", token);
    setIsSignedIn(true);

    // Trigger a storage event so WebSocket can pick up the token
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: "token",
        newValue: token,
        storageArea: localStorage,
      })
    );
  };

  const logout = () => {
    ["token", "user_id", "user_nickname", "user_avatar"].forEach((key) =>
      localStorage.removeItem(key)
    );
    setIsSignedIn(false);

    // Trigger a storage event so WebSocket knows token is cleared
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: "token",
        newValue: null,
        storageArea: localStorage,
      })
    );
  };

  return (
    <AuthContext.Provider value={{ isSignedIn, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use auth
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
};
