import { useState } from "react";
import type { AuthMode } from "../../types";
import type { Credentials } from "./useAuth";

type AuthScreenProps = {
  error: string;
  pending: boolean;
  onSubmit: (credentials: Credentials) => void;
};

export function AuthScreen({ error, pending, onSubmit }: AuthScreenProps) {
  const [mode, setMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  return (
    <main className="auth-screen">
      <div className="auth-mark">JS / CAL</div>
      <section className="auth-panel">
        <p className="eyebrow">A private family calendar</p>
        <h1>
          Make room for
          <br />
          <em>what matters.</em>
        </h1>
        <p className="auth-copy">
          Your days, shared with the people who belong in them.
        </p>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit({ mode, email, password });
          }}
        >
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              minLength={mode === "register" ? 8 : undefined}
              required
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button className="primary-button" type="submit" disabled={pending}>
            {mode === "login" ? "Enter calendar" : "Create account"}{" "}
            <span>↗</span>
          </button>
        </form>
        <button
          className="text-button"
          onClick={() => setMode(mode === "login" ? "register" : "login")}
        >
          {mode === "login"
            ? "New here? Create an account"
            : "Already have an account? Sign in"}
        </button>
        <button
          className="text-button"
          disabled={pending}
          onClick={() => onSubmit({ mode: "demo" })}
        >
          Just looking? Try a demo account
        </button>
        <p className="auth-demo-note">
          No sign-up. You get a sample week, deleted after 24 hours.
        </p>
      </section>
      <div className="auth-note">
        Built for close circles
        <br />
        and ordinary days.
      </div>
    </main>
  );
}
