import { useState } from "react";
import type { FormEvent } from "react";
import type { HouseholdDetails } from "../features/household/useHousehold";
import type { Household } from "../types";

type HouseholdSettingsProps = {
  household: Household | null;
  currentUserId: string;
  onCreate: (details: HouseholdDetails) => Promise<string | null>;
  onUpdate: (details: HouseholdDetails) => Promise<string | null>;
  onAddMember: (email: string) => Promise<string | null>;
  onLeave: () => Promise<string | null>;
};

const readDetails = (form: HTMLFormElement): HouseholdDetails => {
  const data = new FormData(form);
  return {
    name: String(data.get("household-name") ?? ""),
    transit_card_count: Number(data.get("transit-card-count") ?? 1),
  };
};

export function HouseholdSettings({
  household,
  currentUserId,
  onCreate,
  onUpdate,
  onAddMember,
  onLeave,
}: HouseholdSettingsProps) {
  const [message, setMessage] = useState("");
  const run = async (action: Promise<string | null>, success: string) => {
    const error = await action;
    setMessage(error ?? success);
    return error;
  };
  const submitDetails =
    (save: (details: HouseholdDetails) => Promise<string | null>) =>
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      void run(save(readDetails(event.currentTarget)), "Saved.");
    };

  return (
    <section className="settings-section" aria-labelledby="household-heading">
      <h2 id="household-heading">Household</h2>
      <p>
        Members of a household share transit cards and see each other's card
        days.
      </p>
      <form
        className="settings-inline-form"
        key={household?.id ?? "new"}
        onSubmit={submitDetails(household ? onUpdate : onCreate)}
      >
        <label className="settings-select-label" htmlFor="household-name">
          Name
        </label>
        <input
          id="household-name"
          name="household-name"
          defaultValue={household?.name ?? ""}
          placeholder="Family"
          maxLength={60}
          required
        />
        <label className="settings-select-label" htmlFor="transit-card-count">
          Transit cards
        </label>
        <input
          id="transit-card-count"
          name="transit-card-count"
          type="number"
          min={1}
          max={20}
          defaultValue={household?.transit_card_count ?? 1}
          required
        />
        <button className="primary-button compact" type="submit">
          {household ? "Save" : "Create household"}
        </button>
      </form>

      {household && (
        <>
          <ul className="household-members" aria-label="Members">
            {household.members.map((member) => (
              <li key={member.user_id}>
                {member.display_name}
                {member.user_id === currentUserId && " (you)"}
              </li>
            ))}
          </ul>
          <form
            className="settings-inline-form"
            onSubmit={(event) => {
              event.preventDefault();
              const form = event.currentTarget;
              const email = String(
                new FormData(form).get("member-email") ?? "",
              );
              void run(onAddMember(email), "Member added.").then(
                (error) => error ?? form.reset(),
              );
            }}
          >
            <label className="settings-select-label" htmlFor="member-email">
              Add member by email
            </label>
            <input
              id="member-email"
              name="member-email"
              type="email"
              placeholder="name@example.com"
              required
            />
            <button className="primary-button compact" type="submit">
              Add
            </button>
          </form>
          <button
            className="danger-button household-leave"
            type="button"
            onClick={() => void run(onLeave(), "You left the household.")}
          >
            Leave household
          </button>
        </>
      )}
      {message && (
        <p className="form-hint" role="status">
          {message}
        </p>
      )}
    </section>
  );
}
