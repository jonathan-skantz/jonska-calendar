import type { ReactNode } from "react";
import type { WeatherCondition } from "../types";

const LABELS: Record<WeatherCondition, string> = {
  clear: "Sunny",
  "partly-cloudy": "Partly cloudy",
  cloudy: "Cloudy",
  fog: "Fog",
  rain: "Rain",
  snow: "Snow",
  thunder: "Thunder",
};

const CLOUD = "M4.5 12.5a3 3 0 0 1-.4-6 4 4 0 0 1 7.7-.7 3.3 3.3 0 0 1 .2 6.7Z";
const SMALL_CLOUD =
  "M4 10.5a2.3 2.3 0 0 1-.3-4.6 3.1 3.1 0 0 1 5.9-.5 2.6 2.6 0 0 1 .1 5.1Z";

export const Sun = ({ cx, cy, r }: { cx: number; cy: number; r: number }) => (
  <g className="sun">
    <circle cx={cx} cy={cy} r={r} fill="currentColor" />
    {[0, 45, 90, 135, 180, 225, 270, 315].map((angle) => (
      <line
        key={angle}
        x1={cx}
        y1={cy - r - 1}
        x2={cx}
        y2={cy - r - 2.4}
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinecap="round"
        transform={`rotate(${angle} ${cx} ${cy})`}
      />
    ))}
  </g>
);

const shapes: Record<WeatherCondition, ReactNode> = {
  clear: <Sun cx={8} cy={8} r={3.2} />,
  "partly-cloudy": (
    <>
      <Sun cx={10.5} cy={5.5} r={2.2} />
      <path
        className="cloud"
        d={SMALL_CLOUD}
        transform="translate(1 3)"
        fill="currentColor"
      />
    </>
  ),
  cloudy: <path className="cloud" d={CLOUD} fill="currentColor" />,
  fog: (
    <g
      className="cloud"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
    >
      <line x1="2" y1="5" x2="14" y2="5" />
      <line x1="3" y1="8.5" x2="13" y2="8.5" />
      <line x1="2" y1="12" x2="14" y2="12" />
    </g>
  ),
  rain: (
    <>
      <path
        className="cloud"
        d={CLOUD}
        transform="translate(0 -3)"
        fill="currentColor"
      />
      <g
        className="drops"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinecap="round"
      >
        <line x1="5" y1="11.5" x2="4.2" y2="14.5" />
        <line x1="8.5" y1="11.5" x2="7.7" y2="14.5" />
        <line x1="12" y1="11.5" x2="11.2" y2="14.5" />
      </g>
    </>
  ),
  snow: (
    <>
      <path
        className="cloud"
        d={CLOUD}
        transform="translate(0 -3)"
        fill="currentColor"
      />
      <g className="flakes" fill="currentColor">
        <circle cx="4.5" cy="12.5" r="1" />
        <circle cx="8" cy="14" r="1" />
        <circle cx="11.5" cy="12.5" r="1" />
      </g>
    </>
  ),
  thunder: (
    <>
      <path
        className="cloud"
        d={CLOUD}
        transform="translate(0 -3)"
        fill="currentColor"
      />
      <path
        className="bolt"
        d="M8.8 9.5 6.5 12.6h2l-1 3 3.2-4h-2.1l.9-2.1Z"
        fill="currentColor"
      />
    </>
  ),
};

const describeWeather = (conditions: WeatherCondition[]) =>
  conditions.map((condition) => LABELS[condition]).join(", ");

export const WeatherIcons = ({
  conditions,
}: {
  conditions: WeatherCondition[];
}) => (
  <div
    className="day-weather"
    role={conditions.length > 0 ? "img" : undefined}
    aria-label={conditions.length > 0 ? describeWeather(conditions) : undefined}
    title={conditions.length > 0 ? describeWeather(conditions) : undefined}
  >
    {conditions.map((condition) => (
      <svg
        key={condition}
        className={`weather-icon ${condition}`}
        data-condition={condition}
        viewBox="0 0 16 16"
        width="14"
        height="14"
        aria-hidden="true"
        focusable="false"
      >
        {shapes[condition]}
      </svg>
    ))}
  </div>
);
