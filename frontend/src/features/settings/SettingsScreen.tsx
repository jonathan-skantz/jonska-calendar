import type { User } from "../../types";
import { useAuthActions } from "../auth/useAuth";
import { HouseholdSettings } from "../household/HouseholdSettings";
import { useHousehold, useHouseholdMutations } from "../household/useHousehold";
import { IcsSettings } from "../icalendar/IcsSettings";
import { icsExportUrl, useIcsImport } from "../icalendar/useIcsImport";
import { useWorkspace } from "../timeline/useWorkspace";
import { SettingsPage } from "./SettingsPage";

type SettingsScreenProps = {
  user: User;
  onBack: () => void;
};

// The settings page with the profile, household, and tag data it uses.
// Preferences come from PreferencesProvider.
export const SettingsScreen = ({ user, onBack }: SettingsScreenProps) => {
  const { updateDisplayName } = useAuthActions();
  const household = useHousehold().data ?? null;
  const householdMutations = useHouseholdMutations();
  const tags = useWorkspace().data?.tags ?? [];
  const icsImport = useIcsImport();

  return (
    <SettingsPage
      displayName={user.display_name ?? ""}
      displayNamePlaceholder={user.email.split("@")[0]}
      onDisplayNameSave={updateDisplayName}
      onBack={onBack}
    >
      <HouseholdSettings
        household={household}
        currentUserId={user.id}
        onCreate={householdMutations.create}
        onUpdate={householdMutations.update}
        onAddMember={householdMutations.addMember}
        onLeave={householdMutations.leave}
      />
      <IcsSettings
        tags={tags}
        exportUrl={icsExportUrl()}
        onImport={icsImport.mutateAsync}
      />
    </SettingsPage>
  );
};
