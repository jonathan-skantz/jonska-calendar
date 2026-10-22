import { useState } from "react";
import { formatShortDate, formatTime } from "../../lib/calendar";
import type { TimeFormatPreference } from "../../lib/calendar";
import type { Subscription } from "../../types";
import { usePreferences } from "../calendar/preferencesContext";
import type { SubscriptionDetails } from "./useSubscriptions";

const defaultColor = "#547968";

type SubscriptionSettingsProps = {
  subscriptions: Subscription[];
  onAdd: (url: string, details: SubscriptionDetails) => Promise<string | null>;
  onUpdate: (
    id: string,
    details: SubscriptionDetails,
  ) => Promise<string | null>;
  onRefresh: (id: string) => Promise<string | null>;
  onRemove: (id: string) => Promise<string | null>;
};

const formatWhen = (value: string, preference: TimeFormatPreference) =>
  `${formatShortDate(new Date(value))} ${formatTime(value, { preference })}`;

export function SubscriptionSettings({
  subscriptions,
  onAdd,
  onUpdate,
  onRefresh,
  onRemove,
}: SubscriptionSettingsProps) {
  const { timeFormat } = usePreferences().preferences;
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const run = async (action: () => Promise<string | null>, success: string) => {
    setBusy(true);
    setMessage("");
    const error = await action();
    setBusy(false);
    setMessage(error ?? success);
    return error;
  };

  return (
    <section
      className="settings-section"
      aria-labelledby="subscriptions-heading"
    >
      <h2 id="subscriptions-heading">Subscribed calendars</h2>
      <p>
        Follow another calendar by its address, such as a school schedule or
        public holidays. Its events show read-only and are refreshed every few
        hours.
      </p>
      <form
        className="settings-inline-form"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const data = new FormData(form);
          void run(
            () =>
              onAdd(String(data.get("subscription-url") ?? ""), {
                name: String(data.get("subscription-name") ?? ""),
                color: String(data.get("subscription-color") ?? defaultColor),
              }),
            "Subscribed.",
          ).then((error) => error ?? form.reset());
        }}
      >
        <label className="settings-select-label" htmlFor="subscription-url">
          Calendar address
        </label>
        <input
          id="subscription-url"
          name="subscription-url"
          type="text"
          inputMode="url"
          placeholder="webcal://example.com/calendar.ics"
          required
        />
        <label className="settings-select-label" htmlFor="subscription-name">
          Name (optional)
        </label>
        <input
          id="subscription-name"
          name="subscription-name"
          placeholder="The calendar's own name"
          maxLength={100}
        />
        <input
          className="subscription-color-input"
          name="subscription-color"
          type="color"
          defaultValue={defaultColor}
          aria-label="Color"
        />
        <button
          className="primary-button compact"
          type="submit"
          disabled={busy}
        >
          {busy ? "Working…" : "Subscribe"}
        </button>
      </form>

      {subscriptions.length > 0 && (
        <ul className="settings-list" aria-label="Subscriptions">
          {subscriptions.map((subscription) => (
            <li key={subscription.id}>
              <form
                className="settings-inline-form"
                onSubmit={(event) => {
                  event.preventDefault();
                  const data = new FormData(event.currentTarget);
                  void run(
                    () =>
                      onUpdate(subscription.id, {
                        name: String(data.get("name") ?? ""),
                        color: String(data.get("color") ?? subscription.color),
                      }),
                    "Saved.",
                  );
                }}
              >
                <input
                  className="subscription-color-input"
                  name="color"
                  type="color"
                  defaultValue={subscription.color}
                  aria-label={`Color of ${subscription.name}`}
                />
                <input
                  name="name"
                  defaultValue={subscription.name}
                  aria-label={`Name of ${subscription.name}`}
                  maxLength={100}
                  required
                />
                <button className="text-button" type="submit" disabled={busy}>
                  Save
                </button>
                <button
                  className="text-button"
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    void run(() => onRefresh(subscription.id), "Refreshed.")
                  }
                >
                  Refresh now
                </button>
                <button
                  className="text-button"
                  type="button"
                  disabled={busy}
                  onClick={() => {
                    if (
                      window.confirm(`Unsubscribe from “${subscription.name}”?`)
                    )
                      void run(
                        () => onRemove(subscription.id),
                        "Unsubscribed.",
                      );
                  }}
                >
                  Unsubscribe
                </button>
              </form>
              <p className="form-hint settings-list-detail">
                {subscription.url}
                <br />
                {subscription.last_error
                  ? `Last refresh failed (${formatWhen(subscription.last_fetched_at, timeFormat)}): ${subscription.last_error}. Showing events from ${formatWhen(subscription.last_success_at, timeFormat)}.`
                  : `Updated ${formatWhen(subscription.last_success_at, timeFormat)}.`}
              </p>
            </li>
          ))}
        </ul>
      )}
      {message && (
        <p className="form-hint" role="status">
          {message}
        </p>
      )}
    </section>
  );
}
