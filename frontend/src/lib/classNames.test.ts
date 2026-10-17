import { describe, expect, it } from "vitest";
import { cx } from "./classNames";

describe("cx", () => {
  it("joins only the class names that are set", () => {
    expect(cx("grid-entry", false, "event", null, undefined, "")).toBe(
      "grid-entry event",
    );
    expect(cx()).toBe("");
  });
});
