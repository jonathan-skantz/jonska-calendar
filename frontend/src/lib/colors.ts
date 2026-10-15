import { wcagContrast } from "culori";

export const tagTextColor = (hex: string) =>
  wcagContrast(hex, "white") > wcagContrast(hex, "black")
    ? "#ffffff"
    : "#000000";
