import type { ReactNode } from "react";

type WeekHeaderProps = {
  title: string;
  itemCount: number;
  onPrevious: () => void;
  onNext: () => void;
  onAddItem: () => void;
  quickAdd?: ReactNode;
  children?: ReactNode;
};

export const WeekHeader = ({
  title,
  itemCount,
  onPrevious,
  onNext,
  onAddItem,
  quickAdd,
  children,
}: WeekHeaderProps) => (
  <>
    <div className="section-heading week-heading">
      <div className="week-title">
        <button
          className="week-nav"
          onClick={onPrevious}
          aria-label="Previous week"
        >
          ←
        </button>
        <div>
          <span className="section-kicker">Week view</span>
          <h2>{title}</h2>
        </div>
        <button className="week-nav" onClick={onNext} aria-label="Next week">
          →
        </button>
      </div>
      <div className="week-actions">
        {quickAdd}
        <span className="item-count">{itemCount} items</span>
        <button className="primary-button compact" onClick={onAddItem}>
          Add item <span>＋</span>
        </button>
      </div>
    </div>
    {children}
  </>
);
