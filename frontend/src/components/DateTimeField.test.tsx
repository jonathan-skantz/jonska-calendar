import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { DateTimeField } from "./DateTimeField";

afterEach(cleanup);

const value = (container: HTMLElement, name: string) =>
  (container.querySelector(`input[name="${name}"]`) as HTMLInputElement).value;

describe("DateTimeField", () => {
  it("starts timed fields at the day's edges when leaving all-day mode", () => {
    const field = (name: string, dateOnly: boolean) => (
      <DateTimeField
        name={name}
        label={name}
        defaultValue="2026-10-15T10:30"
        preference="24-hour"
        dateOnly={dateOnly}
      />
    );
    const start = render(field("starts_at", true));
    const end = render(field("ends_at", true));

    start.rerender(field("starts_at", false));
    end.rerender(field("ends_at", false));

    expect(value(start.container, "starts_at")).toBe("2026-10-15T00:00");
    expect(value(end.container, "ends_at")).toBe("2026-10-15T23:59");
  });

  it("keeps the time when it starts out timed", () => {
    const { container, rerender } = render(
      <DateTimeField
        name="starts_at"
        label="Starts"
        defaultValue="2026-10-15T10:30"
        preference="24-hour"
      />,
    );
    rerender(
      <DateTimeField
        name="starts_at"
        label="Starts"
        defaultValue="2026-10-15T10:30"
        preference="24-hour"
        dateOnly={false}
      />,
    );
    expect(value(container, "starts_at")).toBe("2026-10-15T10:30");
  });
});
