import { describe, expect, it } from "vitest";
import {
  describeApproximateEdges,
  getEntryBackground,
  getHorizontalInsets,
  getTagStripImage,
  getVerticalInsets,
} from "./entryStyle";

const tag = (id: string, color: string) => ({
  id,
  name: id,
  color,
  is_default: false,
});

describe("tag strip", () => {
  it("fills the strip with the main tag alone", () => {
    expect(getTagStripImage(tag("main", "#111111"), [])).toBe(
      "linear-gradient(to right, #111111 0%, #111111 100%)",
    );
  });

  it("gives other tags equal shares of the right half", () => {
    expect(
      getTagStripImage(tag("main", "#111111"), [
        tag("a", "#aaaaaa"),
        tag("b", "#bbbbbb"),
      ]),
    ).toBe(
      "linear-gradient(to right, #111111 0%, #111111 50%, #aaaaaa 50%, #aaaaaa 75%, #bbbbbb 75%, #bbbbbb 100%)",
    );
  });

  it("draws no strip without tags", () => {
    expect(getTagStripImage(undefined, [])).toBeNull();
  });
});

describe("entry background", () => {
  it("stacks the strip above the approximate-edge fades", () => {
    const background = getEntryBackground("strip", true, true);
    expect(background.backgroundImage).toBe(
      "strip, linear-gradient(to bottom, var(--paper), transparent), linear-gradient(to top, var(--paper), transparent)",
    );
    expect(background.backgroundPosition).toBe("top, top, bottom");
  });

  it("is empty without a strip or fades", () => {
    expect(getEntryBackground(null, false, false).backgroundImage).toBe("");
  });
});

describe("insets", () => {
  it("steps later overlapping blocks in on both sides", () => {
    expect(getHorizontalInsets(2, true)).toEqual({
      left: "15px",
      right: "15px",
    });
  });

  it("insets a block over a ghost on the left only", () => {
    expect(getHorizontalInsets(0, true)).toEqual({
      left: "11px",
      right: undefined,
    });
    expect(getHorizontalInsets(0, false)).toEqual({
      left: undefined,
      right: undefined,
    });
  });

  const vertical = {
    trimTop: true,
    trimBottom: true,
    hasCommuteBefore: false,
    hasCommuteAfter: false,
    hasOptionalGathering: false,
    continuesFromPreviousDay: false,
    continuesToNextDay: false,
  };

  it("gives the adjacent-event separator to the outermost piece", () => {
    expect(getVerticalInsets(vertical)).toEqual({
      outerBandTop: 1,
      top: 1,
      bottom: 1,
    });
    // The optional gathering band sits above the block and takes it.
    expect(
      getVerticalInsets({ ...vertical, hasOptionalGathering: true }),
    ).toEqual({ outerBandTop: 1, top: 0, bottom: 1 });
    // Commute bands take it at both ends.
    expect(
      getVerticalInsets({
        ...vertical,
        hasCommuteBefore: true,
        hasCommuteAfter: true,
      }),
    ).toEqual({ outerBandTop: 0, top: 0, bottom: 0 });
  });

  it("leaves room for continuation markers", () => {
    expect(
      getVerticalInsets({
        ...vertical,
        trimTop: false,
        trimBottom: false,
        continuesFromPreviousDay: true,
        continuesToNextDay: true,
      }),
    ).toEqual({ outerBandTop: 0, top: 6, bottom: 6 });
  });
});

describe("approximate edges", () => {
  it("describes which edges are approximate", () => {
    expect(describeApproximateEdges(true, true)).toBe(
      "Approximate start and end",
    );
    expect(describeApproximateEdges(true, false)).toBe("Approximate start");
    expect(describeApproximateEdges(false, true)).toBe("Approximate end");
    expect(describeApproximateEdges(false, false)).toBeUndefined();
  });
});
