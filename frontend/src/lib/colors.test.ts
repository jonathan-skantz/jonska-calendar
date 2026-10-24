import { describe, expect, it } from "vitest";
import { tagTextColor } from "./colors";

describe("tagTextColor", () => {
  it("picks the text color with more contrast", () => {
    expect(tagTextColor("#1f3d2b")).toBe("#ffffff");
    expect(tagTextColor("#f1c968")).toBe("#000000");
  });
});
