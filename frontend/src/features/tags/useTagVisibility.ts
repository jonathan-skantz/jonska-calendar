import { useState } from "react";

export const useTagVisibility = () => {
  const [hiddenTagIds, setHiddenTagIds] = useState<string[]>([]);

  const toggleTagVisibility = (tagID: string) => {
    setHiddenTagIds((current) =>
      current.includes(tagID)
        ? current.filter((id) => id !== tagID)
        : [...current, tagID],
    );
  };

  const removeHiddenTag = (tagID: string) => {
    setHiddenTagIds((current) => current.filter((id) => id !== tagID));
  };

  return { hiddenTagIds, toggleTagVisibility, removeHiddenTag };
};
