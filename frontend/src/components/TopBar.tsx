type TopBarProps = {
  email: string;
  onOpenSettings: () => void;
  onLogout: () => void;
};

export function TopBar({ email, onOpenSettings, onLogout }: TopBarProps) {
  return (
    <header className="topbar">
      <div className="brand">
        <span className="brand-symbol">✳</span>
        <span>calendar / home</span>
      </div>
      <div className="topbar-actions">
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
