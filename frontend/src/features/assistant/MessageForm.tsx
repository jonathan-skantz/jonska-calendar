import { useState } from "react";
import type { FormEvent } from "react";

const MAX_TEXT_LENGTH = 1000;

type MessageFormProps = {
  pending: boolean;
  // Rejects when the message failed; the text is then put back to retry.
  onSend: (text: string) => Promise<unknown>;
};

// Enter sends; Shift+Enter starts a new line.
export const MessageForm = ({ pending, onSend }: MessageFormProps) => {
  const [text, setText] = useState("");

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const message = text.trim();
    if (!message || pending) return;
    setText("");
    onSend(message).catch(() => setText(message));
  };

  return (
    <form className="assistant-input" onSubmit={submit}>
      <textarea
        aria-label="Message the assistant"
        rows={2}
        maxLength={MAX_TEXT_LENGTH}
        value={text}
        disabled={pending}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            event.currentTarget.form?.requestSubmit();
          }
        }}
      />
      <button
        type="submit"
        className="primary-button"
        disabled={pending || !text.trim()}
      >
        Send
      </button>
    </form>
  );
};
