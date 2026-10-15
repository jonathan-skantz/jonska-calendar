import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EventTagPicker } from "./EventTagPicker";

const tags = [
  { id: "work", name: "Work", color: "#336699", is_default: false },
  { id: "home", name: "Home", color: "#993366", is_default: false },
];

afterEach(cleanup);

describe("EventTagPicker", () => {
  it("attaches a tag when selecting it as the main tag", () => {
    const onSelectedTagIdsChange = vi.fn();
    const onMainTagIdChange = vi.fn();
    render(
      <EventTagPicker
        tags={tags}
        selectedTagIds={["work"]}
        mainTagId="work"
        onSelectedTagIdsChange={onSelectedTagIdsChange}
        onMainTagIdChange={onMainTagIdChange}
      />,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "main" })[1]);

    expect(onMainTagIdChange).toHaveBeenCalledWith("home");
    const updateSelection = onSelectedTagIdsChange.mock.calls[0][0];
    expect(updateSelection(["work"])).toEqual(["work", "home"]);
  });

  it("describes an invalid tag group with the composer error", () => {
    render(
      <EventTagPicker
        tags={tags}
        selectedTagIds={[]}
        mainTagId={null}
        invalid
        errorId="composer-error"
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

    expect(screen.getByRole("group", { name: "Event tags" }).getAttribute("aria-describedby"))
      .toBe("composer-error");
  });
});
