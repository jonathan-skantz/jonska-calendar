import type { Tag } from "../../types";

type TagEditorDrawerProps = {
  tags: Tag[];
  error: string;
  onClose: () => void;
  onUpdate: (tag: Tag, name: string, color: string) => void;
  onUpdateDefault: (tag: Tag) => void;
  onDelete: (tag: Tag) => void;
};

export const TagEditorDrawer = ({
  tags,
  error,
  onClose,
  onUpdate,
  onUpdateDefault,
  onDelete,
}: TagEditorDrawerProps) => (
  <div
    className="drawer-backdrop"
    onMouseDown={(event) => event.target === event.currentTarget && onClose()}
  >
    <aside className="composer drawer tag-editor">
      <div className="modal-heading">
        <div>
          <p className="eyebrow">Event filters</p>
          <h2>Edit tags.</h2>
        </div>
        <button className="icon-button" onClick={onClose} aria-label="Close">
          ×
        </button>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="tag-editor-list">
        {tags.map((tag) => (
          <form
            className="tag-editor-entry"
            key={tag.id}
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              const name = String(form.get("name") || "").trim();
              const color = String(form.get("color") || tag.color);
              onUpdate(tag, name, color);
            }}
          >
            <label>
              Name
              <input name="name" defaultValue={tag.name} required />
            </label>
            <label>
              Color
              <input
                className="tag-color-input"
                name="color"
                type="color"
                defaultValue={tag.color}
              />
            </label>
            <div className="tag-editor-actions">
              <button className="primary-button" type="submit">
                Save
              </button>
              <button
                className="text-button"
                type="button"
                onClick={() => onUpdateDefault(tag)}
              >
                {tag.is_default ? "Default tag" : "Make default"}
              </button>
              <button
                className="danger-button"
                type="button"
                onClick={() => onDelete(tag)}
              >
                Delete
              </button>
            </div>
          </form>
        ))}
      </div>
    </aside>
  </div>
);
