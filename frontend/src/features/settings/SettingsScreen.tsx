import type { User } from "../../types";
import { useAuthActions } from "../auth/useAuth";
import { FeedSettings } from "../feeds/FeedSettings";
import { useFeedMutations, useFeeds } from "../feeds/useFeeds";
import { HouseholdSettings } from "../household/HouseholdSettings";
import { useHousehold, useHouseholdMutations } from "../household/useHousehold";
import { IcsSettings } from "../icalendar/IcsSettings";
import { icsExportUrl, useIcsImport } from "../icalendar/useIcsImport";
import { SubscriptionSettings } from "../subscriptions/SubscriptionSettings";
import {
  useSubscriptionMutations,
  useSubscriptions,
} from "../subscriptions/useSubscriptions";
import { useTags } from "../tags/useTags";
import { SettingsPage } from "./SettingsPage";

type SettingsScreenProps = {
  user: User;
  onBack: () => void;
};

// The settings page with the profile, household, tag, subscription, and feed
// data it uses.
// Preferences come from PreferencesProvider.
export const SettingsScreen = ({ user, onBack }: SettingsScreenProps) => {
  const { updateDisplayName } = useAuthActions();
  const household = useHousehold().data ?? null;
  const householdMutations = useHouseholdMutations();
  const tags = useTags().data ?? [];
  const icsImport = useIcsImport();
  const subscriptions = useSubscriptions().data ?? [];
  const subscriptionMutations = useSubscriptionMutations();
  const feeds = useFeeds().data ?? [];
  const feedMutations = useFeedMutations();
  const isDemo = user.demo_expires_at !== undefined;

  return (
    <SettingsPage
      displayName={user.display_name ?? ""}
      displayNamePlaceholder={user.email.split("@")[0]}
      onDisplayNameSave={updateDisplayName}
      onBack={onBack}
    >
      {/* A demo account cannot reach other users or fetch calendars. */}
      {!isDemo && (
        <HouseholdSettings
          household={household}
          currentUserId={user.id}
          onCreate={householdMutations.create}
          onUpdate={householdMutations.update}
          onAddMember={householdMutations.addMember}
          onLeave={householdMutations.leave}
        />
      )}
      <IcsSettings
        tags={tags}
        exportUrl={icsExportUrl()}
        onImport={icsImport.mutateAsync}
      />
      {!isDemo && (
        <SubscriptionSettings
          subscriptions={subscriptions}
          onAdd={subscriptionMutations.add}
          onUpdate={subscriptionMutations.update}
          onRefresh={subscriptionMutations.refresh}
          onRemove={subscriptionMutations.remove}
        />
      )}
      <FeedSettings
        feeds={feeds}
        tags={tags}
        onCreate={feedMutations.create}
        onUpdate={feedMutations.update}
        onResetUrl={feedMutations.resetUrl}
        onRemove={feedMutations.remove}
      />
    </SettingsPage>
  );
};
