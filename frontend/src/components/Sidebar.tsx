import { tagTextColor } from "../lib/colors";
import type { Tag } from "../types";

type SidebarProps = {
  openTaskCount: number;
  tags: Tag[];
  hiddenTagIds: string[];
  onToggleTagVisibility: (tagId: string) => void;
  onAddTag: () => void;
  onEditTags: () => void;
};

export const Sidebar = ({
  openTaskCount,
  tags,
  hiddenTagIds,
  onToggleTagVisibility,
  onAddTag,
  onEditTags,
}: SidebarProps) => (
  <aside className="side-panel">
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
        <h2>Event filters</h2>
        <button className="icon-button" aria-label="Add tag" onClick={onAddTag}>
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
      <p className="tag-help">
        Click a tag to hide or show its events. Default:{" "}
        {tags.find((tag) => tag.is_default)?.name ?? "none"}
      </p>
    </div>
    <div className="tag-editor-trigger">
      <button className="text-button" onClick={onEditTags}>
        Edit
      </button>
    </div>
    <div className="side-section family-note">
      <span className="note-symbol">✳</span>
      <p>
        Events shared with you are read-only; only the owner can change them.
      </p>
    </div>
  </aside>
);
