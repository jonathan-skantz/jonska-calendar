import type { Tag } from "../../types";

type EventTagPickerProps = {
  tags: Tag[];
  selectedTagIds: string[];
  mainTagId: string | null;
  invalid?: boolean;
  errorId?: string;
  onSelectedTagIdsChange: (update: (current: string[]) => string[]) => void;
  onMainTagIdChange: (tagId: string) => void;
};

export const EventTagPicker = ({
  tags,
  selectedTagIds,
  mainTagId,
  invalid = false,
  errorId,
  onSelectedTagIdsChange,
  onMainTagIdChange,
}: EventTagPickerProps) => (
  <fieldset
    className={invalid ? "tag-picker field-invalid" : "tag-picker"}
    aria-describedby={invalid ? errorId : undefined}
  >
    <legend>Event tags</legend>
    {tags.map((tag) => (
      <div className="tag-option" key={tag.id}>
        <button
          type="button"
          className={
            mainTagId === tag.id ? "main-tag-button active" : "main-tag-button"
          }
          onClick={() => {
            onMainTagIdChange(tag.id);
            onSelectedTagIdsChange((current) =>
              current.includes(tag.id) ? current : [...current, tag.id],
            );
          }}
        >
          main
        </button>
        <label>
          <input
            type="checkbox"
            checked={selectedTagIds.includes(tag.id)}
            onChange={() =>
              onSelectedTagIdsChange((current) =>
                current.includes(tag.id)
                  ? current.filter((id) => id !== tag.id)
                  : [...current, tag.id],
              )
            }
          />
          {tag.name}
        </label>
      </div>
    ))}
  </fieldset>
);
