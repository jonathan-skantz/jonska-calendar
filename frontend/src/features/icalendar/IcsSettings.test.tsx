import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { IcsImportResult, Tag } from "../../types";
import { IcsSettings } from "./IcsSettings";

afterEach(cleanup);

const tags: Tag[] = [
  { id: "tag-1", name: "general", color: "#336699", is_default: true },
  { id: "tag-2", name: "Imported", color: "#996633", is_default: false },
];

const preview: IcsImportResult = {
  imported: 2,
  skipped: [
    { title: "Weekly", reason: "recurring events are not supported yet" },
  ],
  warnings: ["Dinner: unknown time zone"],
};

const renderSettings = (
  onImport: (request: unknown) => Promise<IcsImportResult>,
) =>
  render(<IcsSettings tags={tags} exportUrl="/export" onImport={onImport} />);

const chooseFile = (contents = "BEGIN:VCALENDAR", size?: number) => {
  const file = new File([contents], "calendar.ics", { type: "text/calendar" });
  if (size !== undefined) Object.defineProperty(file, "size", { value: size });
  fireEvent.change(screen.getByLabelText("Import a calendar file"), {
    target: { files: [file] },
  });
};

describe("IcsSettings", () => {
  it("links to the export", () => {
    renderSettings(vi.fn());
    expect(
      screen.getByRole("link", { name: "Download my events (.ics)" }),
    ).toHaveProperty("href", expect.stringContaining("/export"));
  });

  it("previews the import before saving it", async () => {
    const onImport = vi
      .fn()
      .mockResolvedValueOnce(preview)
      .mockResolvedValueOnce({ imported: 2, skipped: [], warnings: [] });
    renderSettings(onImport);

    expect(
      (screen.getByRole("button", { name: "Preview" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    chooseFile();
    fireEvent.change(screen.getByLabelText("Tag imported events with"), {
      target: { value: "tag-2" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    await screen.findByText("2 events will be imported, 1 skipped.");
    expect(onImport).toHaveBeenLastCalledWith({
      ics: "BEGIN:VCALENDAR",
      tagId: "tag-2",
      dryRun: true,
    });
    expect(
      screen.getByText("Weekly: recurring events are not supported yet"),
    ).toBeTruthy();
    expect(screen.getByText("Dinner: unknown time zone")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Import 2 events" }));
    await screen.findByText("Imported 2 events.");
    expect(onImport).toHaveBeenLastCalledWith({
      ics: "BEGIN:VCALENDAR",
      tagId: "tag-2",
      dryRun: false,
    });
    expect(screen.getByRole("button", { name: "Preview" })).toBeTruthy();
  });

  it("asks for a new preview when the file changes", async () => {
    renderSettings(vi.fn().mockResolvedValue(preview));
    chooseFile();
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await screen.findByRole("button", { name: "Import 2 events" });

    chooseFile("BEGIN:VCALENDAR\nother");
    expect(screen.getByRole("button", { name: "Preview" })).toBeTruthy();
  });

  it("does not offer to import nothing", async () => {
    renderSettings(
      vi.fn().mockResolvedValue({ imported: 0, skipped: [], warnings: [] }),
    );
    chooseFile();
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    const button = await screen.findByRole("button", {
      name: "Import 0 events",
    });
    expect((button as HTMLButtonElement).disabled).toBe(true);
  });

  it("shows the server's error", async () => {
    renderSettings(
      vi
        .fn()
        .mockRejectedValue(new Error("the file is not a valid iCalendar file")),
    );
    chooseFile();
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await screen.findByText("the file is not a valid iCalendar file");
  });

  it("rejects files over 5 MB without sending them", async () => {
    const onImport = vi.fn();
    renderSettings(onImport);
    chooseFile("x", 6 * 1024 * 1024);
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() =>
      expect(
        screen.getByText("The file is too large. The limit is 5 MB."),
      ).toBeTruthy(),
    );
    expect(onImport).not.toHaveBeenCalled();
  });
});
