export const BusIcon = ({ crossed = false }: { crossed?: boolean }) => (
  <svg
    className="bus-icon"
    viewBox="0 0 16 16"
    width="11"
    height="11"
    aria-hidden="true"
    focusable="false"
  >
    <path
      fill="currentColor"
      d="M4 1h8a2 2 0 0 1 2 2v9a1 1 0 0 1-1 1v1.5a.5.5 0 0 1-.5.5h-1a.5.5 0 0 1-.5-.5V13H5v1.5a.5.5 0 0 1-.5.5h-1a.5.5 0 0 1-.5-.5V13a1 1 0 0 1-1-1V3a2 2 0 0 1 2-2Zm0 2.5v4h8v-4H4Zm.75 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2Zm6.5 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z"
    />
    {crossed && (
      <path
        d="M1 15 15 1"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    )}
  </svg>
);
