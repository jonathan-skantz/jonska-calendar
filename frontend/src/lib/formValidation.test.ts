import { describe, expect, it } from "vitest";
import { getComposerFieldErrors } from "./formValidation";

const makeForm = (fields: Record<string, string>) => {
  const form = document.createElement("form");
  for (const [name, value] of Object.entries(fields)) {
    const input = document.createElement("input");
    input.name = name;
    input.value = value;
    if (["title", "starts_at", "ends_at", "deadline_at"].includes(name))
      input.required = true;
    form.append(input);
  }
  return form;
};

describe("composer submission validation", () => {
  it("accepts a valid event with an attached main tag", () => {
    const form = makeForm({
      title: "Planning",
      starts_at: "2026-10-15T10:00",
      ends_at: "2026-10-15T11:00",
    });
    expect(getComposerFieldErrors(form, "event", ["work"], "work")).toEqual(
      {},
    );
  });

  it("reports required fields, invalid event time order, and invalid tag selection", () => {
    const form = makeForm({
      title: "",
      starts_at: "2026-10-15T11:00",
      ends_at: "2026-10-15T10:00",
    });
    expect(getComposerFieldErrors(form, "event", [], "other")).toEqual({
      title: true,
      starts_at: true,
      ends_at: true,
      "event-tags": true,
      "main-tag": true,
    });
  });

  it("validates required task fields without applying event tag rules", () => {
    const form = makeForm({ title: "Follow up", deadline_at: "" });
    expect(getComposerFieldErrors(form, "task", [], null)).toEqual({
      deadline_at: true,
    });
  });
});
