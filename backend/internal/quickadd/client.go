package quickadd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const DefaultModel = anthropic.ModelClaudeHaiku5_5

// Client calls the Anthropic Messages API with structured outputs, so the
// reply is JSON matching draftSchema. It is still untrusted and has to pass
// validateDraft before anyone sees it.
type Client struct {
	api   anthropic.Client
	model anthropic.Model
}

// NewClient takes extra options so tests can point it at a fake server.
func NewClient(apiKey, model string, opts ...option.RequestOption) *Client {
	if model == "" {
		model = string(DefaultModel)
	}
	opts = append([]option.RequestOption{
		option.WithAPIKey(apiKey),
		// The first request after a schema change waits for the API to
		// compile it, which is slower than later ones.
		option.WithRequestTimeout(45 * time.Second),
		option.WithMaxRetries(1),
	}, opts...)
	return &Client{api: anthropic.NewClient(opts...), model: anthropic.Model(model)}
}

var outputSchema = func() map[string]any {
	var schema map[string]any
	if err := json.Unmarshal([]byte(draftSchema), &schema); err != nil {
		panic(fmt.Sprintf("quickadd: invalid draft schema: %v", err))
	}
	return schema
}()

// Parse returns the model's raw draft for text. tagNames are the user's own
// tags, offered so the model can map loose wording onto them.
func (c *Client) Parse(ctx context.Context, text string, tagNames []string) (RawDraft, error) {
	message, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt(tagNames)}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(text))},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: outputSchema},
		},
	})
	if err != nil {
		return RawDraft{}, err
	}
	if message.StopReason != anthropic.StopReasonEndTurn {
		return RawDraft{}, fmt.Errorf("anthropic stopped with %q", message.StopReason)
	}
	for _, block := range message.Content {
		if block.Type == "text" {
			var draft RawDraft
			if err := json.Unmarshal([]byte(block.Text), &draft); err != nil {
				return RawDraft{}, fmt.Errorf("decoding draft: %w", err)
			}
			return draft, nil
		}
	}
	return RawDraft{}, errors.New("anthropic reply has no text")
}

func systemPrompt(tagNames []string) string {
	tags := "(none)"
	if len(tagNames) > 0 {
		quoted := make([]string, len(tagNames))
		for i, name := range tagNames {
			quoted[i] = fmt.Sprintf("%q", name)
		}
		tags = strings.Join(quoted, ", ")
	}
	return `You turn one sentence describing a calendar event into a JSON event draft. The sentence may be in Swedish or English.

Rules:
- Record only what the sentence states. Set every field it does not mention to null (lists to []). Never guess or invent values.
- If the sentence mentions something you cannot express exactly in the fields (e.g. "sometime next week", "after lunch", "efter jobbet"), quote that part in "unresolved" instead of guessing.
- Never compute calendar dates. Describe the date as stated:
  - "today"/"idag" = relative_days 0, "tomorrow"/"imorgon" = 1, "the day after tomorrow"/"i övermorgon" = 2, "in 3 days"/"om 3 dagar" = 3.
  - "Friday"/"on Friday"/"this Friday"/"på fredag" = weekday with week_offset 0. "next Friday"/"nästa fredag"/"fredag nästa vecka" = weekday with week_offset 1.
  - "Oct 24"/"24 oktober"/"24/10" = calendar with month and day; include year only if stated.
  - end_date is null unless the event spans several days.
  - In a date object, set the fields its kind does not use to null.
- Times are 24-hour "HH:MM". "kl 18", "6pm", "18" mean "18:00". A range like "18-21" gives start_time and end_time. "for 2 hours"/"i 2 timmar" gives duration_minutes.
- "about", "around", "ca", "cirka", "runt", "ungefär", "~" before a time mark that edge approximate.
- "heads-up", "info", "FYI", releases, birthdays, announcements and deadlines are kind "informational". Things the person takes part in are "attendable". Use null if unclear.
- all_day is true only when the sentence says all day/heldag or the event is clearly a whole-day thing without times (birthday, holiday, release day).
- Commute/pendling/travel/restid before or after gives commute minutes. Gathering/samling/uppvärmning/warm-up/mingel before the start gives gathering_minutes; mingle/mingel/drop-in is optional; put a short name like "Warm-up" or "Mingel" in gathering_label.
- "remind me 1 hour before"/"påminn en timme innan" gives reminders with minutes_before 60. "the day before" is 1440.
- Location: the place as written ("hos mormor" gives "mormor", "at Pizza Hut" gives "Pizza Hut", "on zoom" gives "Zoom").
- Tags: the user's existing tags are: ` + tags + `. Put a tag in "tags" when the sentence names it with "#name" or clearly refers to it. Use the existing tag's exact name when it matches; otherwise write the name as stated.
- title: a short title in the sentence's language, without the date, time, location, tag, commute and reminder words. Keep names of people.`
}

// draftSchema describes the reply. Every field is required but nullable: null
// means "not stated", and the form fills in its default. Optional fields would
// say the same, but each one makes the API's grammar compilation slower; with
// this many it took longer than the request timeout. Value ranges and enums
// are checked in validateDraft rather than here for the same reason.
const draftSchema = `{
  "type": "object",
  "properties": {
    "title": {"type": ["string", "null"]},
    "date": {"anyOf": [{"$ref": "#/$defs/date"}, {"type": "null"}]},
    "end_date": {"anyOf": [{"$ref": "#/$defs/date"}, {"type": "null"}]},
    "start_time": {"type": ["string", "null"], "description": "HH:MM, 24-hour"},
    "end_time": {"type": ["string", "null"], "description": "HH:MM, 24-hour"},
    "duration_minutes": {"type": ["integer", "null"]},
    "all_day": {"type": ["boolean", "null"]},
    "start_is_approximate": {"type": ["boolean", "null"]},
    "end_is_approximate": {"type": ["boolean", "null"]},
    "location": {"type": ["string", "null"]},
    "kind": {"type": ["string", "null"], "description": "attendable or informational"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "commute_before_minutes": {"type": ["integer", "null"]},
    "commute_after_minutes": {"type": ["integer", "null"]},
    "gathering_minutes": {"type": ["integer", "null"]},
    "gathering_is_optional": {"type": ["boolean", "null"]},
    "gathering_label": {"type": ["string", "null"]},
    "reminders": {"type": "array", "items": {"type": "object", "properties": {"minutes_before": {"type": "integer"}}, "required": ["minutes_before"], "additionalProperties": false}},
    "unresolved": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["title", "date", "end_date", "start_time", "end_time", "duration_minutes", "all_day",
    "start_is_approximate", "end_is_approximate", "location", "kind", "tags", "commute_before_minutes",
    "commute_after_minutes", "gathering_minutes", "gathering_is_optional", "gathering_label", "reminders", "unresolved"],
  "additionalProperties": false,
  "$defs": {
    "date": {
      "type": "object",
      "properties": {
        "kind": {"type": "string", "enum": ["relative_days", "weekday", "calendar"]},
        "days": {"type": ["integer", "null"]},
        "weekday": {"type": ["string", "null"], "description": "monday to sunday, lowercase English"},
        "week_offset": {"type": ["integer", "null"], "description": "0 or 1"},
        "month": {"type": ["integer", "null"]},
        "day": {"type": ["integer", "null"]},
        "year": {"type": ["integer", "null"]}
      },
      "required": ["kind", "days", "weekday", "week_offset", "month", "day", "year"],
      "additionalProperties": false
    }
  }
}`
