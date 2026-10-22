import { describe, expect, it } from "vitest";
import {
  customizationFrom,
  customizationPath,
  draftFrom,
} from "./customization";
import type { TimelineEntry } from "../../types";

describe("customization drafts", () => {
  it("round-trips a customization through the form", () => {
    const customization = {
      title: "My lecture",
      note: "Zoom",
      start_shift_seconds: 900,
      end_shift_seconds: -1800,
    };
    const draft = draftFrom(customization);

    expect(draft).toMatchObject({
      title: "My lecture",
      location: "",
      note: "Zoom",
      startShiftMinutes: "15",
      endShiftMinutes: "-30",
    });
    expect(customizationFrom(draft)).toEqual(customization);
  });

  it("keeps the received value for blank text and zero shifts", () => {
    expect(
      customizationFrom({
        ...draftFrom(),
        title: "  ",
        location: " Hall B ",
        startShiftMinutes: "0",
      }),
    ).toEqual({ location: "Hall B" });
  });

  it("refuses shifts that are not whole minutes", () => {
    expect(
      customizationFrom({ ...draftFrom(), startShiftMinutes: "1.5" }),
    ).toBeNull();
    expect(
      customizationFrom({ ...draftFrom(), endShiftMinutes: "soon" }),
    ).toBeNull();
  });

  it("customizes subscribed events by their own route", () => {
    const entry = { id: "e1", type: "event" } as TimelineEntry;
    expect(customizationPath(entry)).toBe("/api/events/e1/customization");
    expect(
      customizationPath({
        ...entry,
        subscription: { id: "s1", name: "School", color: "#123456" },
      }),
    ).toBe("/api/subscribed-events/e1/customization");
  });
});
