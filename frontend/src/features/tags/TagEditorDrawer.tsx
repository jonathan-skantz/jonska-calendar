import type { Tag } from "../../types";
import { Drawer } from "../../components/Drawer";

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
  <Drawer
    eyebrow="Event filters"
    title="Edit tags."
    className="tag-editor"
    onClose={onClose}
  >
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
  </Drawer>
);
