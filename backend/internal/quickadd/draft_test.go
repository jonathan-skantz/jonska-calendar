package quickadd

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

func ptr[T any](value T) *T { return &value }

func TestValidateDraftKeepsValidFields(t *testing.T) {
	result := validateDraft(RawDraft{
		Title:                ptr("  Middag  "),
		Date:                 &DateSpec{Kind: "weekday", Weekday: "saturday", WeekOffset: ptr(1)},
		StartTime:            ptr("18:00"),
		EndTime:              ptr("21:00"),
		StartIsApproximate:   ptr(true),
		Location:             ptr("mormor"),
		Kind:                 ptr("attendable"),
		CommuteBeforeMinutes: ptr(30),
		GatheringMinutes:     ptr(15),
		GatheringLabel:       ptr("Mingel"),
		Reminders:            []RawReminder{{MinutesBefore: 60}},
		Unresolved:           []string{"after work"},
	}, nil)

	draft := result.Draft
	if *draft.Title != "Middag" || draft.Date.Weekday != "saturday" || *draft.StartTime != "18:00" ||
		*draft.EndTime != "21:00" || !*draft.StartIsApproximate || *draft.Location != "mormor" ||
		*draft.Kind != "attendable" || *draft.CommuteBeforeMinutes != 30 || *draft.GatheringMinutes != 15 ||
		*draft.GatheringLabel != "Mingel" || !slices.Equal(draft.ReminderMinutes, []int{60}) {
		t.Fatalf("unexpected draft %+v", draft)
	}
	if len(result.Warnings) != 0 || !slices.Equal(result.Unresolved, []string{"after work"}) {
		t.Fatalf("unexpected warnings %v or unresolved %v", result.Warnings, result.Unresolved)
	}
}

func TestValidateDraftDropsInvalidFields(t *testing.T) {
	result := validateDraft(RawDraft{
		Title:                ptr("   "),
		Date:                 &DateSpec{Kind: "calendar", Month: ptr(13), Day: ptr(1)},
		EndDate:              &DateSpec{Kind: "weekday", Weekday: "fredag"},
		StartTime:            ptr("25:00"),
		EndTime:              ptr("6pm"),
		DurationMinutes:      ptr(0),
		Kind:                 ptr("meeting"),
		CommuteBeforeMinutes: ptr(-5),
		CommuteAfterMinutes:  ptr(5000),
		Reminders:            []RawReminder{{MinutesBefore: -1}, {MinutesBefore: 50000}, {MinutesBefore: 10}},
	}, nil)

	draft := result.Draft
	if draft.Title != nil || draft.Date != nil || draft.EndDate != nil || draft.StartTime != nil ||
		draft.EndTime != nil || draft.DurationMinutes != nil || draft.Kind != nil || draft.CommuteBeforeMinutes != nil {
		t.Fatalf("expected invalid fields to be dropped, got %+v", draft)
	}
	if draft.CommuteAfterMinutes == nil || *draft.CommuteAfterMinutes != maxCommuteMinutes {
		t.Fatalf("expected the commute after to be clamped, got %v", draft.CommuteAfterMinutes)
	}
	if !slices.Equal(draft.ReminderMinutes, []int{10}) {
		t.Fatalf("expected only the valid reminder, got %v", draft.ReminderMinutes)
	}
	// Blank title is absent, not invalid; every other dropped field warns.
	if len(result.Warnings) != 10 {
		t.Fatalf("expected 10 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateDraftMatchesOwnTags(t *testing.T) {
	work, family := uuid.New(), uuid.New()
	tags := []db.Tag{{ID: work, Name: "Work"}, {ID: family, Name: "familj"}}

	result := validateDraft(RawDraft{Tags: []string{"#work", "Familj", "WORK", "gym", " "}}, tags)

	if !slices.Equal(result.Draft.TagIDs, []uuid.UUID{work, family}) {
		t.Fatalf("expected matched tags in order without duplicates, got %v", result.Draft.TagIDs)
	}
	if !slices.Equal(result.Warnings, []string{`No tag named "gym"`}) {
		t.Fatalf("unexpected warnings %v", result.Warnings)
	}
}

func TestValidDate(t *testing.T) {
	cases := []struct {
		spec DateSpec
		want bool
	}{
		{DateSpec{Kind: "relative_days", Days: ptr(1)}, true},
		{DateSpec{Kind: "relative_days", Days: ptr(-1)}, false},
		{DateSpec{Kind: "relative_days"}, false},
		{DateSpec{Kind: "weekday", Weekday: "friday"}, true},
		{DateSpec{Kind: "weekday", Weekday: "friday", WeekOffset: ptr(2)}, false},
		{DateSpec{Kind: "calendar", Month: ptr(10), Day: ptr(24)}, true},
		{DateSpec{Kind: "calendar", Month: ptr(10), Day: ptr(32)}, false},
		{DateSpec{Kind: "calendar", Month: ptr(10), Day: ptr(24), Year: ptr(1999)}, false},
		{DateSpec{Kind: "soon"}, false},
	}
	for _, c := range cases {
		if got := validDate(c.spec); got != c.want {
			t.Errorf("validDate(%+v) = %v, want %v", c.spec, got, c.want)
		}
	}
}

// Optional schema fields slow the API's grammar compilation enough to time
// out, so every property must be required (and nullable when it can be absent).
func TestDraftSchemaRequiresEveryProperty(t *testing.T) {
	var check func(path string, node map[string]any)
	check = func(path string, node map[string]any) {
		if properties, ok := node["properties"].(map[string]any); ok {
			required := map[string]bool{}
			for _, name := range node["required"].([]any) {
				required[name.(string)] = true
			}
			for name, child := range properties {
				if !required[name] {
					t.Errorf("%s.%s is optional", path, name)
				}
				check(path+"."+name, child.(map[string]any))
			}
			if node["additionalProperties"] != false {
				t.Errorf("%s allows additional properties", path)
			}
		}
		if items, ok := node["items"].(map[string]any); ok {
			check(path+"[]", items)
		}
		definitions, _ := node["$defs"].(map[string]any)
		for name, definition := range definitions {
			check("$defs."+name, definition.(map[string]any))
		}
	}
	check("draft", outputSchema)
}
