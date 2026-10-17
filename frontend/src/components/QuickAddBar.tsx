import { useState } from "react";
import type { FormEvent } from "react";

type QuickAddBarProps = {
  // Resolves to an error message, or null once the draft is open.
  onSubmit: (text: string) => Promise<string | null>;
};

export const QuickAddBar = ({ onSubmit }: QuickAddBarProps) => {
  const [text, setText] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!text.trim() || pending) return;
    setPending(true);
    setError(null);
    const message = await onSubmit(text.trim());
    setPending(false);
    // Keep the text on failure so it can be retried or reworded.
    if (message) setError(message);
    else setText("");
  };

  return (
    <form className="quick-add" onSubmit={submit}>
      <input
        aria-label="Quick add event"
        aria-describedby={error ? "quick-add-error" : undefined}
        aria-invalid={Boolean(error)}
        placeholder="Lunch with Anna tomorrow 12–13 #work"
        maxLength={500}
        value={text}
        disabled={pending}
        onChange={(event) => setText(event.target.value)}
      />
      {pending && <span className="quick-add-status">Reading…</span>}
      {error && (
        <p className="error quick-add-error" id="quick-add-error" role="alert">
          {error}
        </p>
      )}
    </form>
  );
};
