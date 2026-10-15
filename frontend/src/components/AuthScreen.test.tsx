import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AuthScreen } from "./AuthScreen";

describe("AuthScreen", () => {
  it("submits controlled credentials and switches between auth modes", () => {
    const onSubmit = vi.fn();
    const onEmailChange = vi.fn();
    const onPasswordChange = vi.fn();
    const onToggleMode = vi.fn();
    const { rerender } = render(
      <AuthScreen
        mode="login"
        email=""
        password=""
        error=""
        onSubmit={onSubmit}
        onEmailChange={onEmailChange}
        onPasswordChange={onPasswordChange}
        onToggleMode={onToggleMode}
      />,
    );

    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: "secret" },
    });
    expect(onEmailChange).toHaveBeenCalledWith("person@example.com");
    expect(onPasswordChange).toHaveBeenCalledWith("secret");

    fireEvent.submit(screen.getByRole("button", { name: /enter calendar/i }).closest("form")!);
    expect(onSubmit).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: /create an account/i }));
    expect(onToggleMode).toHaveBeenCalledOnce();

    rerender(
      <AuthScreen
        mode="register"
        email="person@example.com"
        password="secret"
        error=""
        onSubmit={onSubmit}
        onEmailChange={onEmailChange}
        onPasswordChange={onPasswordChange}
        onToggleMode={onToggleMode}
      />,
    );
    expect(screen.getByRole("button", { name: /create account/i })).toBeTruthy();
  });
});
