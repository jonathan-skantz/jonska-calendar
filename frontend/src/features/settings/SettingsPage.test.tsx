import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PreferencesProvider } from "../calendar/PreferencesProvider";
import { SettingsPage } from "./SettingsPage";

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

const renderSettings = () =>
  render(
    <PreferencesProvider userId="user-1">
      <SettingsPage
        displayName=""
        displayNamePlaceholder="person"
        onDisplayNameSave={vi.fn()}
        onBack={vi.fn()}
      />
    </PreferencesProvider>,
  );

describe("SettingsPage preferences", () => {
  it("saves changed preferences for the signed-in user", () => {
    renderSettings();

    fireEvent.click(screen.getByLabelText("24-hour"));
    fireEvent.change(screen.getByLabelText("First day of the week"), {
      target: { value: "0" },
    });

    expect((screen.getByLabelText("24-hour") as HTMLInputElement).checked).toBe(
      true,
    );
    expect(
      JSON.parse(
        window.localStorage.getItem("calendar-preferences:user-1") ?? "{}",
      ),
    ).toMatchObject({ timeFormat: "24-hour", weekStartsOn: 0 });
  });
});
