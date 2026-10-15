type TopBarProps = {
  email: string;
  onLogout: () => void;
};

export function TopBar({ email, onLogout }: TopBarProps) {
  return (
    <header className="topbar">
      <div className="brand">
        <span className="brand-symbol">✳</span>
        <span>calendar / home</span>
      </div>
      <div className="topbar-actions">
        <span className="user-email">{email}</span>
        <button className="text-button" onClick={onLogout}>
          Sign out
        </button>
      </div>
    </header>
  );
}
