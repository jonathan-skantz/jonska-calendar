import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TopBar } from "./TopBar";

const renderTopBar = () => {
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
});
