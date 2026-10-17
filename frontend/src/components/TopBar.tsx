type TopBarProps = {
  email: string;
  upcomingReminderCount: number;
  sidebarOpen: boolean;
  onToggleSidebar: () => void;
  onOpenReminders: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
};

export function TopBar({
  email,
  upcomingReminderCount,
  sidebarOpen,
  onToggleSidebar,
  onOpenReminders,
  onOpenSettings,
  onLogout,
}: TopBarProps) {
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
      </div>
      <div className="topbar-actions">
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
          <button className="text-button" onClick={onOpenSettings}>
            Settings
          </button>
          <span className="user-email">{email}</span>
          <button className="text-button" onClick={onLogout}>
            Sign out
          </button>
        </div>
      </div>
    </header>
  );
}
