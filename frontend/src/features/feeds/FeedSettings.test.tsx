import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Feed, Tag } from "../../types";
import { FeedSettings } from "./FeedSettings";
import { feedUrl, webcalUrl } from "./useFeeds";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const tags: Tag[] = [
  { id: "tag-1", name: "general", color: "#336699", is_default: true },
  { id: "tag-2", name: "Work", color: "#996633", is_default: false },
];

const workFeed: Feed = {
  id: "feed-1",
  name: "Colleagues",
  tag_id: "tag-2",
  include_alarms: false,
  path: "/api/feeds/secret/calendar.ics",
};

const renderSettings = (feeds: Feed[]) => {
  const props = {
    feeds,
    tags,
    onCreate: vi.fn().mockResolvedValue(null),
    onUpdate: vi.fn().mockResolvedValue(null),
    onResetUrl: vi.fn().mockResolvedValue(null),
    onRemove: vi.fn().mockResolvedValue(null),
  };
  render(<FeedSettings {...props} />);
  return props;
};

describe("feed addresses", () => {
  it("adds the origin and offers webcal", () => {
    const url = feedUrl(workFeed, "https://cal.example.com");
    expect(url).toBe("https://cal.example.com/api/feeds/secret/calendar.ics");
    expect(webcalUrl(url)).toBe(
      "webcal://cal.example.com/api/feeds/secret/calendar.ics",
    );
    expect(webcalUrl("http://localhost:5173/a.ics")).toBe(
      "webcal://localhost:5173/a.ics",
    );
  });
});

describe("FeedSettings", () => {
  it("creates a feed for one tag with alarms", async () => {
    const { onCreate } = renderSettings([]);
    fireEvent.change(screen.getByLabelText("Feed name"), {
      target: { value: "Colleagues" },
    });
    fireEvent.change(screen.getByLabelText("Events"), {
      target: { value: "tag-2" },
    });
    fireEvent.click(screen.getByLabelText("Include my reminders as alarms"));
    fireEvent.click(screen.getByRole("button", { name: "Create feed" }));

    await screen.findByText("Feed created.");
    expect(onCreate).toHaveBeenCalledWith({
      name: "Colleagues",
      tagId: "tag-2",
      includeAlarms: true,
    });
  });

  it("shows the address and manages the feed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const { onUpdate, onResetUrl, onRemove } = renderSettings([workFeed]);
    const item = within(screen.getByRole("list", { name: "Feeds" }));

    expect(item.getByText("Events tagged Work")).toBeTruthy();
    expect(
      (item.getByLabelText("Address of Colleagues") as HTMLInputElement).value,
    ).toBe(`${window.location.origin}/api/feeds/secret/calendar.ics`);
    expect(
      item.getByRole("link", { name: "Open in calendar app" }),
    ).toHaveProperty("href", expect.stringMatching(/^webcal:/));

    fireEvent.click(item.getByLabelText("Include my reminders as alarms"));
    await screen.findByText("Saved.");
    expect(onUpdate).toHaveBeenCalledWith("feed-1", {
      name: "Colleagues",
      tagId: "tag-2",
      includeAlarms: true,
    });

    fireEvent.click(item.getByRole("button", { name: "Reset address" }));
    await screen.findByText("Address reset.");
    expect(onResetUrl).toHaveBeenCalledWith("feed-1");

    fireEvent.click(item.getByRole("button", { name: "Delete" }));
    await screen.findByText("Feed deleted.");
    expect(onRemove).toHaveBeenCalledWith("feed-1");
  });

  it("does not reset the address without confirmation", () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const { onResetUrl } = renderSettings([workFeed]);
    fireEvent.click(screen.getByRole("button", { name: "Reset address" }));
    expect(onResetUrl).not.toHaveBeenCalled();
  });
});
