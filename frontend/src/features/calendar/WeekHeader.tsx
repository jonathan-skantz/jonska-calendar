import type { ReactNode } from "react";
import { cx } from "../../lib/classNames";
import { VISIBLE_HOUR_OPTIONS } from "./calendarPreferences";
import { usePreferences } from "./preferencesContext";

type WeekHeaderProps = {
  title: string;
  onPrevious: () => void;
  onToday: () => void;
  onNext: () => void;
  onAddItem: () => void;
  quickAdd?: ReactNode;
  children?: ReactNode;
};

export const WeekHeader = ({
  title,
  onPrevious,
  onToday,
  onNext,
  onAddItem,
  quickAdd,
  children,
}: WeekHeaderProps) => {
  const { preferences, updatePreferences } = usePreferences();
  return (
    <>
      <div className="section-heading week-heading">
        <div className="week-title">
          <div className="week-nav-group">
            <button
              className="week-nav"
              onClick={onPrevious}
              aria-label="Previous week"
            >
              ←
            </button>
            <button className="week-nav today-button" onClick={onToday}>
              Today
            </button>
            <button
              className="week-nav"
              onClick={onNext}
              aria-label="Next week"
            >
              →
            </button>
          </div>
          <h2>{title}</h2>
          <div className="zoom-toggle" role="group" aria-label="Zoom">
            {VISIBLE_HOUR_OPTIONS.map((hours) => (
              <button
                key={hours}
                type="button"
                className={cx(preferences.visibleHours === hours && "selected")}
                aria-pressed={preferences.visibleHours === hours}
                title={`Fit ${hours} hours on screen`}
                onClick={() => updatePreferences({ visibleHours: hours })}
              >
                {hours}h
              </button>
            ))}
          </div>
        </div>
        <div className="week-actions">
          {quickAdd}
          <button
            className="primary-button new-item-button"
            onClick={onAddItem}
          >
            <span aria-hidden="true">＋</span> New
          </button>
        </div>
      </div>
      {children}
    </>
  );
};
