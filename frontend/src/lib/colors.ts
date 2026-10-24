import {
  modeLrgb,
  modeRgb,
  useMode as registerMode,
  wcagContrast,
} from "culori/fn";

// The tree-shakable entry starts with no color spaces, so only the two that
// contrast needs end up in the bundle: sRGB to parse colors, and linear RGB
// for luminance.
registerMode(modeRgb);
registerMode(modeLrgb);

export const tagTextColor = (hex: string) =>
  wcagContrast(hex, "white") > wcagContrast(hex, "black")
    ? "#ffffff"
    : "#000000";
