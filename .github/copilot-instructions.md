# Project instructions

## Product specification

`PRODUCT.md` is the source of truth for the intended product behavior.

Before implementing features or changing existing behavior:

* Read the relevant parts of `PRODUCT.md`.
* Understand how the requested change fits into the overall product.
* Preserve the intended relationships and behavior between events, tasks, and other concepts.
* Consider how the implementation will support other described use cases.

## Technical decisions

* The technology stack and architecture are set. See `docs/decisions.md`. Change them only with a written reason.
* Do not assume a particular framework, database, or deployment model without justification.
* Inspect the existing code and follow established decisions where they exist.
* Prefer simple, maintainable, testable designs that leave room for the product to evolve.
* Avoid introducing abstractions or dependencies without a concrete need.
* Do not implement unrelated features just because they are described in the product specification.

## Code comments

* Add concise comments for non-obvious code, especially relationships,
  derived values, authorization boundaries, and database queries that may be
  difficult for a beginner to understand.
* Do not add comments that merely restate obvious code.

## Handling ambiguity

* Distinguish explicitly stated requirements from assumptions and proposed solutions.
* If a product requirement is ambiguous and materially affects the implementation, ask for clarification.
* If multiple technical approaches are reasonable, explain the trade-offs before making a consequential choice.
* If a requested change conflicts with the product specification, identify the conflict before proceeding.

## Implementation workflow

For significant features:

1. Identify the relevant product requirements and use cases.
2. Inspect the existing implementation.
3. Propose a minimal implementation approach, including important trade-offs.
4. Implement the feature and appropriate tests.
5. Identify any assumptions or product decisions that remain unresolved.


## Communication

* Be concise. Use plain, direct language.
* Prefer short bullets over paragraphs.
* Omit filler, repetition, and obvious explanations.
* Explain only non-obvious decisions and trade-offs.
* Keep plans and summaries brief.
* Do not restate the task or existing context.
* Ask questions only when ambiguity materially affects implementation.
* Do not sacrifice code quality or correctness for brevity.

## Token efficiency

* Minimize unnecessary context and output.
* Read only relevant sections of `PRODUCT.md` and other files.
* Avoid repeatedly summarizing the entire project.
* Do not dump entire files, logs, or code unless needed.
* Make focused changes; avoid unrelated refactoring.
* Run targeted tests first.
* Report only changes made, test results, and important caveats.
* Do not generate documentation or plans unless useful or requested.

### Example of the intended style

Instead of:

> I have analyzed the requirements and determined that the best approach would be to introduce a dedicated service layer. This would help improve separation of concerns and make the application easier to maintain in the long run.

Prefer:

> Add a service layer to separate business logic from UI code. This improves testability.


# Simple English

---
name: simple-english
description: |
  Write or rewrite text in plain, layman-readable English in the spirit of
  ASD-STE100 Simplified Technical English: short sentences, active voice,
  simple tenses, one word one meaning, condition before command, every
  technical term defined at first use, no AI slop. Default mode is Plain.
  Strict mode applies full STE vocabulary compliance when the user names
  STE, ASD-STE100, or compliance. Use for documentation, READMEs, runbooks,
  procedures, error messages, release notes, incident reports, API guides,
  and explanations for readers outside the field. Also use when the user
  says "STE", "Simplified Technical English", "ASD-STE100", "plain English",
  "layman's terms", "explain it simply", "no jargon", "de-slop", "make this
  readable", "write for non-native readers", or asks for docs that translate
  well. The same rules govern the reply: answer first, prose only.
license: MIT
compatibility: claude-code cursor codex gemini-cli opencode
metadata:
  version: "2.1.1"
  standard: ASD-STE100 Issue 9 (2025-01-15)
---


Write plain English that a smart reader outside your field understands on one read. The rules come from ASD-STE100, the controlled language that aerospace uses for maintenance manuals. Two registers exist: the document you write or rewrite, and the reply you type in chat. Each has its own short rule set below.

## The Document

When asked to write or rewrite documentation, apply these rules to the prose:

1. Classify each passage. Procedural text tells the reader what to do: imperative mood, 20 words per sentence, one instruction per sentence. Descriptive text explains: simple tenses, 25 words per sentence, one topic per paragraph, six sentences per paragraph at most.
2. Never touch code, identifiers, commands, flags, file paths, quoted errors, product names, or facts. When the source gives no number or cause, keep the general statement.
3. Condition before command, with a comma: "If the build fails, read the log."
4. Use simple tenses and active voice. No present perfect ("has completed" → "completed"). No "-ing" verb after a comma (", making it easy" → new sentence). Name the actor: "You run the migration."
5. Modals: can, will, must. Never should, would, may, might, could. A required "should" becomes "must". An optional one is deleted.
6. Use complete grammar. No contractions, keep articles, keep "that". Write short sentences, not telegraph style.
7. No semicolons and no em-dashes. Write two sentences, or name the relation.
8. One word, one meaning, for the whole document. Use `make sure that` for check, verify, confirm, validate, ensure. Use `configuration` for config, settings, options. Break noun chains over three words with a preposition ("the timeout value for the connection pool").
9. Give the reader each term and each fact before the step that needs it. Define a concept term at its first use, under ten words, one per sentence. Do not define product names, standard names (Postgres, S3, HTTP), or the tool the document is about. Also name the host, the flag, or the prior step that a command depends on. "Restart the service" becomes "Restart the `sync` service on the host that runs the job."
10. State the fact, not its importance. Delete words that carry no fact: simply, seamlessly, robust, powerful, comprehensive, leverage, crucial, "in order to", "it is worth noting". No "not just X, it is Y". No decorative triplets. No "in conclusion".
11. Use formatting only where it carries structure. No bold lead-ins, no bold as emphasis, no emoji, no heading over two sentences. A vertical list is for three or more parallel items or steps: colon on the lead-in, uppercase start, one instruction per item.
12. Warnings: command or condition first, then the risk. "Do not run this against production. The command deletes rows."

Use American spelling. `references/word-swaps.md` maps the overused words to plain ones. For an error message, a runbook, an incident report, release notes, a commit message, or UI copy, read `references/use-cases.md` first. It names the mode and the pattern for each.

Before (AI output):

> **Connection timeouts.** If sqlpipe hangs or fails with `dial tcp: i/o timeout`, check that the host running sqlpipe can reach the Postgres port (usually 5432) — this is often a security group or firewall rule blocking the connection. If you're connecting to a managed database (RDS, Cloud SQL, etc.), confirm the instance allows connections from sqlpipe's IP.

After (procedural, headed, numbered):

> ## Connection timeouts
>
> sqlpipe stops with `dial tcp: i/o timeout` when it cannot connect to the Postgres port (5432 by default).
>
> 1. Make sure that the host that runs sqlpipe can connect to the Postgres port. A firewall or security group usually blocks it.
> 2. If the database is managed (RDS, Cloud SQL), make sure that the instance accepts connections from the IP of sqlpipe.

## The Reply

Every chat reply, in every mode, follows these rules:

1. Answer in prose. No headers, no bullet lists, no bold, no tables. A code block is legal when the reader must copy it.
2. The first sentence gives the answer or the result. Do not restate the question.
3. No em-dashes. Name the relation ("because", "but", "for example") or write two sentences.
4. Define a concept term in a few words the first time you use it: "idempotent (safe to run twice)". Do not define product names.
5. No contractions. No openers ("Certainly", "Great question") and no closers ("I hope this helps", "Let me know").
6. Do not shorten quoted error text, security warnings, or confirmations before a destructive action.

Before: The failure stems from control-plane leader election during pod churn — nothing to worry about!
After: The pods restarted and the queue lost its leader for a short time. It recovered without help. You do not have to do anything.

## Self-Check Before You Deliver

1. Reply: search for `—`, `**`, `#`, and a line that starts with `-`. Remove each one.
2. Document: count the words in your three longest sentences. Over 20 or 25, split. Search for `'`, `has been`, `should`, `may`, `;`, `—`, `, making`, `**`, `check`, `verify`, `config`, and any heading that covers fewer than three sentences. Fix each hit. Read each step: does it name a host, a flag, or a prior step the reader must already have? If not, add it or point to it.

## Modes

Plain is the default and is all of the above. Strict applies when the user names STE, ASD-STE100, or compliance: read `references/strict-vocabulary.md` before you draft the document, and say once that no tool guarantees compliance. The reply stays Plain in every mode.

When asked to CHECK text instead of writing it, first open `references/rule-catalog.md`. Then report each violation as: rule number quoted from that file, the offending text, a compliant rewrite. Never cite a rule number from memory. When the user asked for compliance, end with one sentence: no tool can guarantee ASD-STE100 compliance, and the standard is a free download at asd-ste100.org.

## Limits

These rules are for facts and instructions, not marketing copy or brand writing, because they delete persuasion. Say so, and offer them for the docs instead.

## References

- `references/rule-catalog.md`: the 53 rules of Issue 9 with software examples, for CHECK mode
- `references/strict-vocabulary.md`: the dictionary discipline for Strict mode
- `references/word-swaps.md`: a map from overused words to plain ones
- `references/use-cases.md`: the mode and the pattern for error messages, runbooks, incident reports, release notes, commits, agent prompts, UI copy, and translation prep