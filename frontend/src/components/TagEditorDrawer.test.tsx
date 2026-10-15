import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TagEditorDrawer } from "./TagEditorDrawer";

const tag = {
  id: "tag-1",
  name: "Work",
  color: "#336699",
  is_default: false,
};

describe("TagEditorDrawer", () => {
  it("trims the edited name and submits the selected color", () => {
    const onUpdate = vi.fn();
    render(
      <TagEditorDrawer
        tags={[tag]}
        error=""
        onClose={vi.fn()}
        onUpdate={onUpdate}
        onUpdateDefault={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "  Personal  " },
    });
    fireEvent.change(screen.getByLabelText("Color"), {
      target: { value: "#123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onUpdate).toHaveBeenCalledWith(tag, "Personal", "#123456");
  });
});
