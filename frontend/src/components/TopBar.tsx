import { format } from "date-fns";
import { useRef } from "react";

type TopBarProps = {
  email: string;
  displayName?: string;
  // Set for a demo account: when it and its data are deleted.
  demoExpiresAt?: string;
  upcomingReminderCount: number;
  sidebarOpen: boolean;
  onToggleSidebar: () => void;
  onOpenAssistant: () => void;
  onOpenReminders: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
};

export function TopBar({
  email,
  displayName,
  demoExpiresAt,
  upcomingReminderCount,
  sidebarOpen,
  onToggleSidebar,
  onOpenAssistant,
  onOpenReminders,
  onOpenSettings,
  onLogout,
}: TopBarProps) {
  const accountMenu = useRef<HTMLDivElement>(null);
  // The menu is a native popover: it closes on Escape and outside clicks by
  // itself, and is closed by hand before a menu action runs.
  const chooseFromMenu = (action: () => void) => () => {
    accountMenu.current?.hidePopover?.();
    action();
  };
  return (
    <header className="topbar">
      <div className="topbar-start">
        <button
          className="text-button sidebar-toggle"
          onClick={onToggleSidebar}
          aria-label={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
          aria-expanded={sidebarOpen}
          aria-controls="side-panel"
        >
          ☰
        </button>
        <div className="brand">
          <span className="brand-symbol">✳</span>
          <span className="brand-name">calendar / home</span>
        </div>
        {demoExpiresAt && <span className="demo-badge">Demo</span>}
      </div>
      <div className="topbar-actions">
        <button className="text-button" onClick={onOpenAssistant}>
          Assistant
        </button>
        <button
          className="text-button reminders-button"
          onClick={onOpenReminders}
          aria-label={`Reminders, ${upcomingReminderCount} upcoming`}
        >
          🔔
          {upcomingReminderCount > 0 && (
            <span className="reminder-count">{upcomingReminderCount}</span>
          )}
        </button>
        <div className="topbar-account">
          <button
            className="account-button"
            popoverTarget="account-menu"
            aria-label={`Account menu for ${email}`}
          >
            <span className="account-initial" aria-hidden="true">
              {(displayName || email).slice(0, 1).toUpperCase()}
            </span>
            <span aria-hidden="true">▾</span>
          </button>
          <div
            id="account-menu"
            className="account-menu"
            popover="auto"
            ref={accountMenu}
          >
            <span className="user-email">{email}</span>
            {demoExpiresAt && (
              <span className="user-email">
                Demo account, deleted with its data on{" "}
                {format(new Date(demoExpiresAt), "EEE d MMM 'at' HH:mm")}
              </span>
            )}
            <button
              className="text-button"
              onClick={chooseFromMenu(onOpenSettings)}
            >
              Settings
            </button>
            <button className="text-button" onClick={chooseFromMenu(onLogout)}>
              Sign out
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
