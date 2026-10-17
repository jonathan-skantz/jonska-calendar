import type { FormEvent } from "react";
import type { AuthMode } from "../../types";

type AuthScreenProps = {
  mode: AuthMode;
  email: string;
  password: string;
  error: string;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onToggleMode: () => void;
};

export function AuthScreen({
  mode,
  email,
  password,
  error,
  onSubmit,
  onEmailChange,
  onPasswordChange,
  onToggleMode,
}: AuthScreenProps) {
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
        <form onSubmit={onSubmit}>
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => onEmailChange(event.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(event) => onPasswordChange(event.target.value)}
              required
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button className="primary-button" type="submit">
            {mode === "login" ? "Enter calendar" : "Create account"}{" "}
            <span>↗</span>
          </button>
        </form>
        <button className="text-button" onClick={onToggleMode}>
          {mode === "login"
            ? "New here? Create an account"
            : "Already have an account? Sign in"}
        </button>
      </section>
      <div className="auth-note">
        Built for close circles
        <br />
        and ordinary days.
      </div>
    </main>
  );
}
