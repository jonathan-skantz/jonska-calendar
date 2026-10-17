import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TransitCardDrawer } from "./TransitCardDrawer";
import { getTransitCardDayViews } from "./transitCardDays";

const dates = [19, 20, 21, 22, 23, 24, 25].map((day) => new Date(2026, 9, day));

const renderDrawer = (onSave = vi.fn()) => {
  const views = getTransitCardDayViews(
    dates,
    [],
    new Date(2026, 9, 21, 9),
    "user-1",
  );
  render(
    <TransitCardDrawer
      views={views}
      error=""
      onClose={vi.fn()}
      onSave={onSave}
    />,
  );
  return onSave;
};

describe("TransitCardDrawer", () => {
  afterEach(cleanup);

  it("submits an empty selection as an explicit report", () => {
    const onSave = renderDrawer();

    fireEvent.click(
      screen.getByRole("button", { name: "Save – no days needed" }),
    );

    expect(onSave).toHaveBeenCalledWith({
      from: "2026-10-21",
      to: "2026-10-25",
      needed_dates: [],
      maybe_dates: [],
    });
  });

  it("disables days that have ended and submits yes and maybe answers", () => {
    const onSave = renderDrawer();

    const yesButtons = screen.getAllByRole<HTMLButtonElement>("button", {
      name: "Yes",
    });
    expect(yesButtons.map((button) => button.disabled)).toEqual([
      true,
      true,
      false,
      false,
      false,
      false,
      false,
    ]);
    fireEvent.click(yesButtons[3]);
    fireEvent.click(screen.getAllByRole("button", { name: "Maybe" })[4]);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith({
      from: "2026-10-21",
      to: "2026-10-25",
      needed_dates: ["2026-10-22"],
      maybe_dates: ["2026-10-23"],
    });
  });
});

describe("TransitCardDrawer with a household", () => {
  afterEach(cleanup);

  it("shows other members' answers and warns before overbooking", () => {
    const views = getTransitCardDayViews(
      dates,
      [
        {
          date: "2026-10-22",
          user_id: "user-2",
          display_name: "Anna",
          need: "yes",
          status: "resolved",
        },
      ],
      new Date(2026, 9, 21, 9),
      "user-1",
      [
        { user_id: "user-1", display_name: "Jonathan" },
        { user_id: "user-2", display_name: "Anna" },
      ],
    );
    render(
      <TransitCardDrawer
        views={views}
        cardCount={1}
        error=""
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );

    const thursday = screen.getByRole("group", { name: "Thu, Oct 22" });
    expect(
      thursday.querySelector(".transit-card-pill.resolved")?.textContent,
    ).toBe("Anna");
    expect(thursday.textContent).toContain("1 of 1 card needed");
    expect(screen.queryByRole("alert")).toBeNull();

    fireEvent.click(screen.getAllByRole("button", { name: "Yes" })[3]);

    expect(screen.getByRole("alert").textContent).toBe(
      "2 need it, but there is only 1 card",
    );
  });
});
