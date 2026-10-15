import { useEffect, useState } from "react";
import { CLOCK_UPDATE_INTERVAL_MS } from "../lib/calendar";

export const useCurrentTime = () => {
  const [currentTime, setCurrentTime] = useState(() => new Date());

  useEffect(() => {
    const clock = window.setInterval(
      () => setCurrentTime(new Date()),
      CLOCK_UPDATE_INTERVAL_MS,
    );
    return () => window.clearInterval(clock);
  }, []);

  return currentTime;
};
