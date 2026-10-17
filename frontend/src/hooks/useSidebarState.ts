import { useCallback, useState } from "react";
import { SIDEBAR_STORAGE_KEY } from "../lib/layout";

const readRailOpen = () => {
  try {
    return window.localStorage.getItem(SIDEBAR_STORAGE_KEY) !== "false";
  } catch {
    return true;
  }
};

const storeRailOpen = (open: boolean) => {
  try {
    window.localStorage.setItem(SIDEBAR_STORAGE_KEY, String(open));
  } catch {
    // The choice still applies for this session.
  }
};

// The desktop rail remembers whether it was open; the phone drawer always
// starts closed so the calendar is visible first.
export const useSidebarState = (isDrawer: boolean) => {
  const [railOpen, setRailOpen] = useState(readRailOpen);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const setOpen = useCallback(
    (open: boolean) => {
      if (isDrawer) {
        setDrawerOpen(open);
      } else {
        setRailOpen(open);
        storeRailOpen(open);
      }
    },
    [isDrawer],
  );

  const open = isDrawer ? drawerOpen : railOpen;
  const toggle = useCallback(() => setOpen(!open), [open, setOpen]);
  const close = useCallback(() => setOpen(false), [setOpen]);

  return { open, toggle, close };
};
