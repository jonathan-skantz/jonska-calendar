import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { QuickAddBar } from "./QuickAddBar";

afterEach(cleanup);

const typeAndSubmit = (text: string) => {
  const input = screen.getByLabelText("Quick add event") as HTMLInputElement;
  fireEvent.change(input, { target: { value: text } });
  fireEvent.submit(input.form!);
  return input;
};

describe("QuickAddBar", () => {
  it("submits the trimmed text and clears it once the draft opens", async () => {
    const onSubmit = vi.fn().mockResolvedValue(null);
    render(<QuickAddBar onSubmit={onSubmit} />);

    const input = typeAndSubmit("  lunch tomorrow 12  ");

    expect(onSubmit).toHaveBeenCalledWith("lunch tomorrow 12");
    await waitFor(() => expect(input.value).toBe(""));
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("keeps the text and shows the error when parsing fails", async () => {
    const onSubmit = vi.fn().mockResolvedValue("Quick add is not configured");
    render(<QuickAddBar onSubmit={onSubmit} />);

    const input = typeAndSubmit("lunch tomorrow 12");

    expect((await screen.findByRole("alert")).textContent).toBe(
      "Quick add is not configured",
    );
    expect(input.value).toBe("lunch tomorrow 12");
  });

  it("ignores blank input", () => {
    const onSubmit = vi.fn();
    render(<QuickAddBar onSubmit={onSubmit} />);
    typeAndSubmit("   ");
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
