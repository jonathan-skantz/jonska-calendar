import { useState } from "react";
import type { ChangeEvent } from "react";
import type { ShareFieldsUpdate } from "../features/sharing/useEventShares";
import type { EventShare, ShareFields, TimelineEntry } from "../types";

type ShareEventDrawerProps = {
  event: TimelineEntry;
  shares?: EventShare[];
  onAdd: (email: string) => Promise<string | null>;
  onUpdate: (
    shareId: string,
    fields: ShareFieldsUpdate,
  ) => Promise<string | null>;
  onRemove: (share: EventShare) => Promise<string | null>;
  onClose: () => void;
};

const FIELDS: {
  name: keyof ShareFields;
  label: string;
  multiline?: boolean;
}[] = [
  { name: "title", label: "Title" },
  { name: "location", label: "Location" },
  { name: "url", label: "URL" },
  { name: "description", label: "Details", multiline: true },
];

const customFieldCount = (share: ShareFields) =>
  FIELDS.filter(({ name }) => share[name]).length;

const draftFrom = (share: ShareFields) =>
  Object.fromEntries(
    FIELDS.map(({ name }) => [name, share[name] ?? ""]),
  ) as Record<keyof ShareFields, string>;

type RecipientCardProps = {
  event: TimelineEntry;
  share: EventShare;
  expanded: boolean;
  onToggle: () => void;
  onSave: (fields: ShareFieldsUpdate) => Promise<string | null>;
  onRemove: () => void;
};

const RecipientCard = ({
  event,
  share,
  expanded,
  onToggle,
  onSave,
  onRemove,
}: RecipientCardProps) => {
  const [draft, setDraft] = useState(() => draftFrom(share));
  const custom = customFieldCount(share);
  const fieldsId = `share-fields-${share.id}`;

  return (
    <li className="share-recipient">
      <div className="share-recipient-header">
        <button
          type="button"
          className="share-recipient-toggle"
          aria-expanded={expanded}
          aria-controls={fieldsId}
          onClick={onToggle}
        >
          <span aria-hidden="true">{expanded ? "▾" : "▸"}</span>
          <strong>{share.recipient_display_name}</strong>
          <small>{share.recipient_email}</small>
        </button>
        <span className={`share-badge ${custom ? "custom" : ""}`}>
          {custom ? `${custom} custom` : "Same as event"}
        </span>
        <button
          type="button"
          className="icon-button"
          aria-label={`Stop sharing with ${share.recipient_display_name}`}
          onClick={onRemove}
        >
          ×
        </button>
      </div>
      {expanded && (
        <form
          id={fieldsId}
          className="share-recipient-fields"
          onSubmit={async (formEvent) => {
            formEvent.preventDefault();
            const fields = Object.fromEntries(
              FIELDS.map(({ name }) => [name, draft[name].trim() || null]),
            ) as ShareFieldsUpdate;
            await onSave(fields);
          }}
        >
          {FIELDS.map(({ name, label, multiline }) => {
            const props = {
              name,
              value: draft[name],
              placeholder: event[name] || "Not set",
              onChange: (
                changeEvent: ChangeEvent<
                  HTMLInputElement | HTMLTextAreaElement
                >,
              ) =>
                setDraft((current) => ({
                  ...current,
                  [name]: changeEvent.target.value,
                })),
            };
            return (
              <label key={name}>
                {label}
                {multiline ? (
                  <textarea {...props} />
                ) : (
                  <input {...props} type={name === "url" ? "url" : "text"} />
                )}
              </label>
            );
          })}
          <button className="primary-button compact" type="submit">
            Save for {share.recipient_display_name}
          </button>
        </form>
      )}
    </li>
  );
};

export const ShareEventDrawer = ({
  event,
  shares,
  onAdd,
  onUpdate,
  onRemove,
  onClose,
}: ShareEventDrawerProps) => {
  const [error, setError] = useState("");
  // Keyed by email, which is unique per event and known before a new share's
  // id arrives, so a just-added recipient opens once the list refreshes.
  const [expandedEmail, setExpandedEmail] = useState<string | null>(null);

  return (
    <div
      className="drawer-backdrop share-drawer-backdrop"
      onMouseDown={(mouseEvent) =>
        mouseEvent.target === mouseEvent.currentTarget && onClose()
      }
    >
      <aside className="composer drawer share-drawer">
        <div className="modal-heading">
          <div>
            <p className="eyebrow">Share event</p>
            <h2>Share “{event.title}”</h2>
          </div>
          <button className="icon-button" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
        <form
          className="share-add"
          onSubmit={async (formEvent) => {
            formEvent.preventDefault();
            const form = formEvent.currentTarget;
            const email = String(new FormData(form).get("recipient-email"))
              .trim()
              .toLowerCase();
            const addError = await onAdd(email);
            setError(addError ?? "");
            if (!addError) {
              form.reset();
              setExpandedEmail(email);
            }
          }}
        >
          <label>
            Add recipient
            <input
              name="recipient-email"
              type="email"
              placeholder="name@example.com"
              required
            />
          </label>
          <button className="primary-button compact" type="submit">
            Add
          </button>
        </form>
        <p className="form-hint">
          Leave a field empty to show the event’s own value. Changes to the
          event still reach everyone. Tags are never shared.
        </p>
        {shares && shares.length === 0 && (
          <p className="form-hint">Not shared with anyone yet.</p>
        )}
        <ul className="share-recipients">
          {shares?.map((share) => (
            <RecipientCard
              // Saved values are part of the key so the draft resets to what
              // the server stored after each save.
              key={`${share.id}:${FIELDS.map(({ name }) => share[name] ?? "").join("\u0000")}`}
              event={event}
              share={share}
              expanded={expandedEmail === share.recipient_email}
              onToggle={() =>
                setExpandedEmail((current) =>
                  current === share.recipient_email
                    ? null
                    : share.recipient_email,
                )
              }
              onSave={async (fields) => {
                const saveError = await onUpdate(share.id, fields);
                setError(saveError ?? "");
                return saveError;
              }}
              onRemove={async () => {
                setError((await onRemove(share)) ?? "");
              }}
            />
          ))}
        </ul>
      </aside>
    </div>
  );
};
