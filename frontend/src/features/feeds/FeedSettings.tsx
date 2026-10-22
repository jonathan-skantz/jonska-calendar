import { useState } from "react";
import type { Feed, Tag } from "../../types";
import type { FeedDetails } from "./useFeeds";
import { feedUrl, webcalUrl } from "./useFeeds";

type FeedSettingsProps = {
  feeds: Feed[];
  tags: Tag[];
  onCreate: (details: FeedDetails) => Promise<string | null>;
  onUpdate: (id: string, details: FeedDetails) => Promise<string | null>;
  onResetUrl: (id: string) => Promise<string | null>;
  onRemove: (id: string) => Promise<string | null>;
};

const detailsOf = (feed: Feed): FeedDetails => ({
  name: feed.name,
  tagId: feed.tag_id ?? "",
  includeAlarms: feed.include_alarms,
});

// The feed's address is a secret: anyone who has it sees the events, so it
// can be reset to stop old subscribers.
export function FeedSettings({
  feeds,
  tags,
  onCreate,
  onUpdate,
  onResetUrl,
  onRemove,
}: FeedSettingsProps) {
  const [message, setMessage] = useState("");
  const run = async (action: Promise<string | null>, success: string) => {
    const error = await action;
    setMessage(error ?? success);
    return error;
  };
  const tagName = (id?: string) => tags.find((tag) => tag.id === id)?.name;

  const copy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url);
      setMessage("Address copied.");
    } catch {
      setMessage("Unable to copy; select the address and copy it instead.");
    }
  };

  return (
    <section className="settings-section" aria-labelledby="feeds-heading">
      <h2 id="feeds-heading">Publish my calendar</h2>
      <p>
        Let other calendar apps subscribe to your own events, all of them or
        those with one tag. Anyone with the address can see the events, so share
        it only with people you trust. Events shared with you and subscribed
        calendars are not included.
      </p>
      <form
        className="settings-inline-form"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const data = new FormData(form);
          void run(
            onCreate({
              name: String(data.get("feed-name") ?? ""),
              tagId: String(data.get("feed-tag") ?? ""),
              includeAlarms: data.get("feed-alarms") === "on",
            }),
            "Feed created.",
          ).then((error) => error ?? form.reset());
        }}
      >
        <label className="settings-select-label" htmlFor="feed-name">
          Feed name
        </label>
        <input
          id="feed-name"
          name="feed-name"
          placeholder="Family"
          maxLength={100}
          required
        />
        <label className="settings-select-label" htmlFor="feed-tag">
          Events
        </label>
        <select id="feed-tag" name="feed-tag" defaultValue="">
          <option value="">All my events</option>
          {tags.map((tag) => (
            <option key={tag.id} value={tag.id}>
              Tagged {tag.name}
            </option>
          ))}
        </select>
        <label className="settings-option">
          <input type="checkbox" name="feed-alarms" />
          Include my reminders as alarms
        </label>
        <button className="primary-button compact" type="submit">
          Create feed
        </button>
      </form>

      {feeds.length > 0 && (
        <ul className="settings-list" aria-label="Feeds">
          {feeds.map((feed) => {
            const url = feedUrl(feed);
            return (
              <li key={feed.id}>
                <strong>{feed.name}</strong>
                <p className="form-hint settings-list-detail">
                  {feed.tag_id
                    ? `Events tagged ${tagName(feed.tag_id) ?? "…"}`
                    : "All my events"}
                </p>
                <input
                  className="feed-url"
                  readOnly
                  value={url}
                  aria-label={`Address of ${feed.name}`}
                  onFocus={(event) => event.currentTarget.select()}
                />
                <div className="settings-inline-form">
                  <label className="settings-option">
                    <input
                      type="checkbox"
                      checked={feed.include_alarms}
                      onChange={(event) =>
                        void run(
                          onUpdate(feed.id, {
                            ...detailsOf(feed),
                            includeAlarms: event.target.checked,
                          }),
                          "Saved.",
                        )
                      }
                    />
                    Include my reminders as alarms
                  </label>
                  <button
                    className="text-button"
                    type="button"
                    onClick={() => void copy(url)}
                  >
                    Copy address
                  </button>
                  <a className="text-button" href={webcalUrl(url)}>
                    Open in calendar app
                  </a>
                  <button
                    className="text-button"
                    type="button"
                    onClick={() => {
                      if (
                        window.confirm(
                          `Reset the address of “${feed.name}”? Calendars that subscribe to the old address stop getting updates.`,
                        )
                      )
                        void run(onResetUrl(feed.id), "Address reset.");
                    }}
                  >
                    Reset address
                  </button>
                  <button
                    className="text-button"
                    type="button"
                    onClick={() => {
                      if (window.confirm(`Delete the feed “${feed.name}”?`))
                        void run(onRemove(feed.id), "Feed deleted.");
                    }}
                  >
                    Delete
                  </button>
                </div>
              </li>
            );
          })}
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
