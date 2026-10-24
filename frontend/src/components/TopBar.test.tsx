import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TopBar } from "./TopBar";

const renderTopBar = (demoExpiresAt?: string) => {
  const handlers = {
    onToggleSidebar: vi.fn(),
    onOpenAssistant: vi.fn(),
    onOpenReminders: vi.fn(),
    onOpenSettings: vi.fn(),
    onLogout: vi.fn(),
  };
  render(
    <TopBar
      email="ana@example.com"
      displayName="Ana"
      demoExpiresAt={demoExpiresAt}
      upcomingReminderCount={0}
      sidebarOpen
      {...handlers}
    />,
  );
  return handlers;
};

describe("TopBar", () => {
  afterEach(cleanup);

  it("puts settings and sign out in the account menu", () => {
    const handlers = renderTopBar();
    const button = screen.getByRole("button", {
      name: "Account menu for ana@example.com",
    });
    expect(button.textContent).toContain("A");
    expect(button.getAttribute("popovertarget")).toBe("account-menu");

    // The closed popover is hidden; jsdom does not open it on click.
    fireEvent.click(
      screen.getByRole("button", { name: "Settings", hidden: true }),
    );
    expect(handlers.onOpenSettings).toHaveBeenCalledOnce();
    fireEvent.click(
      screen.getByRole("button", { name: "Sign out", hidden: true }),
    );
    expect(handlers.onLogout).toHaveBeenCalledOnce();
  });

  it("marks a demo account and says when it is deleted", () => {
    renderTopBar("2026-10-25T17:00:00Z");
    expect(screen.getByText("Demo")).toBeTruthy();
    expect(
      screen.getByText(/Demo account, deleted with its data on/),
    ).toBeTruthy();
  });

  it("shows no demo badge for a regular account", () => {
    renderTopBar();
    expect(screen.queryByText("Demo")).toBeNull();
  });
});
