import React, { useState } from "react";

declare global {
  interface ImportMeta {
    env: {
      VITE_API_URL?: string;
      [key: string]: any;
    };
  }
}

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

interface LoginProps {
  onLoginSuccess: () => void;
  onSwitchToRegister: () => void;
}

const Login: React.FC<LoginProps> = ({ onLoginSuccess, onSwitchToRegister }) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      const res = await fetch(`${API_URL}/api/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
        credentials: "include",
      });
      if (res.ok) {
        onLoginSuccess();
      } else {
        setError("Invalid username or password");
      }
    } catch {
      setError("Server error");
    }
  };

  return (
    <div style={{ textAlign: "center", marginTop: 40 }}>
      <form onSubmit={handleLogin} style={{ display: "inline-block" }}>
        <div>
          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={e => setUsername(e.target.value)}
            required
          />
        </div>
        <div style={{ marginTop: 10 }}>
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
          />
        </div>
        <div style={{ marginTop: 20 }}>
          <button type="submit">Login</button>
        </div>
        {error && <div style={{ color: "red", marginTop: 10 }}>{error}</div>}
      </form>
      <div style={{ marginTop: 20 }}>
        <button onClick={onSwitchToRegister}>Create Account</button>
      </div>
    </div>
  );
};

export default Login;
