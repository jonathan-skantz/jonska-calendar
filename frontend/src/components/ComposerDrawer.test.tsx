import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ComposerDrawer } from "./ComposerDrawer";

describe("ComposerDrawer validation state", () => {
  it("exposes invalid fields to assistive technology", () => {
    render(
      <ComposerDrawer
        type="event"
        editingItem={null}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        fieldErrors={{ title: true, starts_at: true, ends_at: true }}
        onSubmit={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Title").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByLabelText("Starts").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByLabelText("Ends").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByLabelText("Description").getAttribute("aria-invalid")).toBe("false");
  });
});
