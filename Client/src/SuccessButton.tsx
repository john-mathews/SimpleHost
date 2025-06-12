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

function SuccessButton() {
  const [message, setMessage] = useState("");

  const getSuccessMessage = async () => {
    try {
      const res = await fetch(`${API_URL}/api/success-message`, {
        credentials: "include",
      });
      const data = await res.json();
      setMessage(data.message);
    } catch (err) {
      setMessage("Error fetching message");
    }
  };

  return (
    <div style={{ textAlign: "center", marginTop: 40 }}>
      <button onClick={getSuccessMessage}>Get Happy Success Message</button>
      <div style={{ marginTop: 20, fontSize: 24 }}>{message}</div>
    </div>
  );
}

export default SuccessButton;
