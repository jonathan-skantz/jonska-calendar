import { useEffect } from "react";
import type { ReactNode } from "react";
import { tagTextColor } from "../lib/colors";
import type { SubscriptionRef, Tag } from "../types";

type SidebarProps = {
  open: boolean;
  // "rail" sits beside the calendar; "drawer" slides over it (phones).
  variant: "rail" | "drawer";
  onClose: () => void;
  // Extra content shown only in the drawer, e.g. account actions that the
  // top bar hides on small screens.
  drawerFooter?: ReactNode;
  openTaskCount: number;
  tags: Tag[];
  subscriptions: SubscriptionRef[];
  // The hidden tags and subscriptions; both filter events the same way.
  hiddenTagIds: string[];
  onToggleTagVisibility: (tagId: string) => void;
  onAddTag: () => void;
  onEditTags: () => void;
};

export const Sidebar = ({
  open,
  variant,
  onClose,
  drawerFooter,
  openTaskCount,
  tags,
  subscriptions,
  hiddenTagIds,
  onToggleTagVisibility,
  onAddTag,
  onEditTags,
}: SidebarProps) => {
  const isOpenDrawer = open && variant === "drawer";

  useEffect(() => {
    if (!isOpenDrawer) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [isOpenDrawer, onClose]);

  if (!open) return null;

  const filterHelp = `Click a tag to hide or show its events. Default tag: ${
    tags.find((tag) => tag.is_default)?.name ?? "none"
  }. Events shared with you are read-only; only the owner can change them.`;
  const filterButton = (
    item: { id: string; name: string; color: string },
    label: string,
  ) => {
    const isHidden = hiddenTagIds.includes(item.id);
    return (
      <div className="tag-control" key={item.id}>
        <button
          className={isHidden ? "tag large hidden" : "tag large"}
          style={{
            backgroundColor: item.color,
            color: tagTextColor(item.color),
          }}
          onClick={() => onToggleTagVisibility(item.id)}
          aria-label={`${isHidden ? "Show" : "Hide"} ${label}`}
        >
          <span
            className={`eye-icon ${isHidden ? "closed" : "open"}`}
            aria-hidden="true"
          />
          {item.name}
        </button>
      </div>
    );
  };

  return (
    <>
      {isOpenDrawer && (
        <div
          className="drawer-backdrop side-panel-backdrop"
          onClick={onClose}
        />
      )}
      <aside id="side-panel" className={`side-panel ${variant}`}>
        <div className="pulse-panel">
          <span className="pulse-dot" />
          <div>
            <p className="eyebrow">Today</p>
            <strong>
              {openTaskCount} open task{openTaskCount === 1 ? "" : "s"}
            </strong>
          </div>
        </div>
        <div className="side-section">
          <div className="section-heading small">
            <h2>
              Event filters{" "}
              <span
                className="info-hint"
                role="img"
                aria-label={filterHelp}
                title={filterHelp}
              >
                ⓘ
              </span>
            </h2>
            <button
              className="icon-button"
              aria-label="Add tag"
              onClick={onAddTag}
            >
              ＋
            </button>
          </div>
          <div className="tag-cloud">
            {tags.length ? (
              tags.map((tag) => filterButton(tag, `${tag.name} events`))
            ) : (
              <span className="muted">No tags yet</span>
            )}
          </div>
        </div>
        <div className="tag-editor-trigger">
          <button className="text-button" onClick={onEditTags}>
            Edit
          </button>
        </div>
        {subscriptions.length > 0 && (
          <div className="side-section">
            <div className="section-heading small">
              <h2>Subscribed calendars</h2>
            </div>
            <div className="tag-cloud">
              {subscriptions.map((subscription) =>
                filterButton(subscription, `events from ${subscription.name}`),
              )}
            </div>
          </div>
        )}
        {variant === "drawer" && drawerFooter}
      </aside>
    </>
  );
};
