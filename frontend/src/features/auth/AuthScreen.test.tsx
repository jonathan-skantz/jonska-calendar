import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthScreen } from "./AuthScreen";

afterEach(cleanup);

const fillCredentials = () => {
  fireEvent.change(screen.getByLabelText("Email"), {
    target: { value: "person@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), {
    target: { value: "secret" },
  });
};

describe("AuthScreen", () => {
  it("submits the entered credentials in the selected mode", () => {
    const onSubmit = vi.fn();
    render(<AuthScreen error="" pending={false} onSubmit={onSubmit} />);

    fillCredentials();
    fireEvent.click(screen.getByRole("button", { name: /enter calendar/i }));
    expect(onSubmit).toHaveBeenLastCalledWith({
      mode: "login",
      email: "person@example.com",
      password: "secret",
    });

    fireEvent.click(screen.getByRole("button", { name: /create an account/i }));
    fireEvent.click(screen.getByRole("button", { name: /create account/i }));
    expect(onSubmit).toHaveBeenLastCalledWith({
      mode: "register",
      email: "person@example.com",
      password: "secret",
    });
  });

  it("starts a demo without credentials", () => {
    const onSubmit = vi.fn();
    render(<AuthScreen error="" pending={false} onSubmit={onSubmit} />);

    fireEvent.click(screen.getByRole("button", { name: /try a demo/i }));
    expect(onSubmit).toHaveBeenLastCalledWith({ mode: "demo" });
  });

  it("disables submitting while signing in and shows errors", () => {
    render(
      <AuthScreen
        error="Invalid email or password"
        pending
        onSubmit={vi.fn()}
      />,
    );

    expect(
      screen
        .getByRole("button", { name: /enter calendar/i })
        .hasAttribute("disabled"),
    ).toBe(true);
    expect(screen.getByText("Invalid email or password")).toBeTruthy();
  });
});
