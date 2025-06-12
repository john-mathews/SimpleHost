import React, { useState } from "react";
import SuccessButton from "./SuccessButton";
import Login from "./Login";
import Register from "./Register";

function App() {
  const [loggedIn, setLoggedIn] = useState(false);
  const [showRegister, setShowRegister] = useState(false);

  if (loggedIn) return <SuccessButton />;
  if (showRegister)
    return (
      <Register
        onRegisterSuccess={() => setShowRegister(false)}
        onSwitchToLogin={() => setShowRegister(false)}
      />
    );
  return (
    <Login
      onLoginSuccess={() => setLoggedIn(true)}
      onSwitchToRegister={() => setShowRegister(true)}
    />
  );
}

export default App;
