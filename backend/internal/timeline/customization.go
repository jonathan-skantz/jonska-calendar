package timeline

import "time"

// ReceivedValues are a received event's values as the owner or the feed
// gives them to the recipient, before the recipient's customization. They
// never include the owner's own values of fields set per recipient.
type ReceivedValues struct {
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	URL         *string   `json:"url,omitempty"`
	Location    *string   `json:"location,omitempty"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
}

// customize applies the recipient's customization of a received event and
// keeps what they received as Original. The shifts are left out on all-day
// events, whose edges are midnights, and when the shifted event would not
// end after it starts, as after the owner shortened it.
func customize(item *TimelineItem) {
	custom := item.Customization
	if custom == nil || item.StartsAt == nil || item.EndsAt == nil {
		return
	}
	item.Original = &ReceivedValues{
		Title: item.Title, Description: item.Description, URL: item.URL, Location: item.Location,
		StartsAt: *item.StartsAt, EndsAt: *item.EndsAt,
	}
	if custom.Title != nil {
		item.Title = *custom.Title
	}
	item.Description = customized(custom.Description, item.Description)
	item.URL = customized(custom.URL, item.URL)
	item.Location = customized(custom.Location, item.Location)
	if item.IsAllDay {
		return
	}
	startsAt := item.StartsAt.Add(time.Duration(custom.StartShiftSeconds) * time.Second)
	endsAt := item.EndsAt.Add(time.Duration(custom.EndShiftSeconds) * time.Second)
	if endsAt.After(startsAt) {
		item.StartsAt, item.EndsAt = &startsAt, &endsAt
	}
}

func customized(custom, received *string) *string {
	if custom != nil {
		return custom
	}
	return received
}
