import { useEffect } from "react";
import type { ReactNode } from "react";
import { tagTextColor } from "../lib/colors";
import type { Tag } from "../types";

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
              tags.map((tag) => {
                const isHidden = hiddenTagIds.includes(tag.id);
                return (
                  <div className="tag-control" key={tag.id}>
                    <button
                      className={isHidden ? "tag large hidden" : "tag large"}
                      style={{
                        backgroundColor: tag.color,
                        color: tagTextColor(tag.color),
                      }}
                      onClick={() => onToggleTagVisibility(tag.id)}
                      aria-label={`${isHidden ? "Show" : "Hide"} ${tag.name} events`}
                    >
                      <span
                        className={`eye-icon ${isHidden ? "closed" : "open"}`}
                        aria-hidden="true"
                      />
                      {tag.name}
                    </button>
                  </div>
                );
              })
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
        {variant === "drawer" && drawerFooter}
      </aside>
    </>
  );
};
