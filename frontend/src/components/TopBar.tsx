type TopBarProps = {
  email: string;
  upcomingReminderCount: number;
  onOpenReminders: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
};

export function TopBar({
  email,
  upcomingReminderCount,
  onOpenReminders,
  onOpenSettings,
  onLogout,
}: TopBarProps) {
  return (
    <header className="topbar">
      <div className="brand">
        <span className="brand-symbol">✳</span>
        <span>calendar / home</span>
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
        <button className="text-button" onClick={onOpenSettings}>
          Settings
        </button>
        <span className="user-email">{email}</span>
        <button className="text-button" onClick={onLogout}>
          Sign out
        </button>
      </div>
    </header>
  );
}
