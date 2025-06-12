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

interface RegisterProps {
  onRegisterSuccess: () => void;
  onSwitchToLogin: () => void;
}

const Register: React.FC<RegisterProps> = ({ onRegisterSuccess, onSwitchToLogin }) => {
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [retypePassword, setRetypePassword] = useState("");
  const [error, setError] = useState("");

  const validatePassword = (pw: string) => {
    // At least 8 chars, 1 uppercase, 1 lowercase, 1 number, 1 special char
    return /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]).{8,}$/.test(pw);
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!validatePassword(password)) {
      setError("Password must be at least 8 characters and include uppercase, lowercase, number, and special character.");
      return;
    }
    if (password !== retypePassword) {
      setError("Passwords do not match.");
      return;
    }
    try {
      const res = await fetch(`${API_URL}/api/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, username, password }),
      });
      if (res.status === 201) {
        onRegisterSuccess();
      } else if (res.status === 409) {
        setError("Username or email already exists.");
      } else {
        setError("Registration failed.");
      }
    } catch {
      setError("Server error");
    }
  };

  return (
    <div style={{ textAlign: "center", marginTop: 40 }}>
      <form onSubmit={handleRegister} style={{ display: "inline-block" }}>
        <div>
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            required
          />
        </div>
        <div style={{ marginTop: 10 }}>
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
        <div style={{ marginTop: 10 }}>
          <input
            type="password"
            placeholder="Retype Password"
            value={retypePassword}
            onChange={e => setRetypePassword(e.target.value)}
            required
          />
        </div>
        <div style={{ marginTop: 20 }}>
          <button type="submit">Create Account</button>
        </div>
        {error && <div style={{ color: "red", marginTop: 10 }}>{error}</div>}
      </form>
      <div style={{ marginTop: 20 }}>
        <button onClick={onSwitchToLogin}>Already have an account? Login</button>
      </div>
    </div>
  );
};

export default Register;
