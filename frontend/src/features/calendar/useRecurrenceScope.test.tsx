import {
  act,
  cleanup,
  fireEvent,
  render,
  renderHook,
  screen,
} from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { useRecurrenceScope } from "./useRecurrenceScope";

afterEach(cleanup);

const setup = () => {
  const hook = renderHook(() => useRecurrenceScope());
  const showDialog = () => render(<>{hook.result.current.dialog}</>);
  return { hook, showDialog };
};

describe("useRecurrenceScope", () => {
  it("answers with the chosen occurrences", async () => {
    const { hook, showDialog } = setup();
    let answer: Promise<string | null> = Promise.resolve(null);
    act(() => {
      answer = hook.result.current.ask({ action: "move", allowThis: true });
    });
    showDialog();
    expect(
      screen.getByRole("dialog", { name: "Move repeating event" }),
    ).toBeTruthy();
    fireEvent.click(
      screen.getByRole("button", { name: "This and following events" }),
    );
    await expect(answer).resolves.toBe("following");
    expect(hook.result.current.dialog).toBeNull();
  });

  it("leaves out this event for changes one occurrence cannot have", async () => {
    const { hook, showDialog } = setup();
    let answer: Promise<string | null> = Promise.resolve("all");
    act(() => {
      answer = hook.result.current.ask({ action: "save", allowThis: false });
    });
    showDialog();
    expect(screen.queryByRole("button", { name: "This event" })).toBeNull();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    await expect(answer).resolves.toBeNull();
  });
});
