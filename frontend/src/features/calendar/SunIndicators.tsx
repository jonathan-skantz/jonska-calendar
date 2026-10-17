import { Sun } from "../../components/WeatherIcons";
import type {
  SunIndicators as SunIndicatorsState,
  SunLine,
} from "../weather/sunTimes";

const Line = ({
  kind,
  line,
}: {
  kind: "sunrise" | "sunset";
  line: SunLine;
}) => (
  <div className={`sun-line ${kind}`} style={{ top: line.top }}>
    <span className="sun-line-label">
      <svg viewBox="0 0 16 16" width="12" height="12" focusable="false">
        <Sun cx={8} cy={8} r={3.2} />
      </svg>
      {kind === "sunrise" ? "↑" : "↓"} {line.label}
    </span>
  </div>
);

export const SunIndicators = ({
  sunrise,
  sunset,
  night,
}: SunIndicatorsState) => (
  <>
    <div className="sun-indicators night-layer" aria-hidden="true">
      {night.map((band) => (
        <div
          className="night-overlay"
          key={band.top}
          style={{ top: band.top, height: band.height }}
        />
      ))}
    </div>
    <div className="sun-indicators line-layer" aria-hidden="true">
      {sunrise && <Line kind="sunrise" line={sunrise} />}
      {sunset && <Line kind="sunset" line={sunset} />}
    </div>
  </>
);
