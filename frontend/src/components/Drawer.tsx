import type { ReactNode } from "react";
import { cx } from "../lib/classNames";

type DrawerProps = {
  eyebrow: ReactNode;
  title: ReactNode;
  // Added to the drawer panel, next to the shared "composer drawer" classes.
  className?: string;
  // Stacked drawers raise their backdrop above the drawer they open from.
  backdropClassName?: string;
  onClose: () => void;
  children: ReactNode;
};

// A side drawer over a backdrop. Pressing the backdrop itself closes it;
// presses that start inside the panel do not.
export const Drawer = ({
  eyebrow,
  title,
  className,
  backdropClassName,
  onClose,
  children,
}: DrawerProps) => (
  <div
    className={cx("drawer-backdrop", backdropClassName)}
    onMouseDown={(event) => event.target === event.currentTarget && onClose()}
  >
    <aside className={cx("composer drawer", className)}>
      <div className="modal-heading">
        <div>
          <p className="eyebrow">{eyebrow}</p>
          <h2>{title}</h2>
        </div>
        <button className="icon-button" onClick={onClose} aria-label="Close">
          ×
        </button>
      </div>
      {children}
    </aside>
  </div>
);
