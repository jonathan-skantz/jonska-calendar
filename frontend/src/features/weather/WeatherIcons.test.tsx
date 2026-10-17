import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { WeatherIcons } from "./WeatherIcons";

describe("WeatherIcons", () => {
  afterEach(cleanup);

  it("shows one icon per condition with a combined label", () => {
    const { container } = render(
      <WeatherIcons conditions={["clear", "rain"]} />,
    );
    expect(screen.getByRole("img", { name: "Sunny, Rain" })).toBeTruthy();
    const icons = container.querySelectorAll(".weather-icon");
    expect(
      [...icons].map((icon) => icon.getAttribute("data-condition")),
    ).toEqual(["clear", "rain"]);
  });

  it("renders an empty placeholder when there is no forecast", () => {
    const { container } = render(<WeatherIcons conditions={[]} />);
    expect(screen.queryByRole("img")).toBeNull();
    expect(container.querySelectorAll(".weather-icon")).toHaveLength(0);
  });
});
