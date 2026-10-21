import { useState } from "react";
import { getRequestErrorMessage } from "../../lib/requestErrors";
import type { IcsImportResult, Tag } from "../../types";
import type { IcsImportRequest } from "./useIcsImport";

// Matches the server's request limit, leaving room for JSON escaping.
const maxFileBytes = 5 * 1024 * 1024;

type IcsSettingsProps = {
  tags: Tag[];
  exportUrl: string;
  onImport: (request: IcsImportRequest) => Promise<IcsImportResult>;
};

const eventCount = (count: number) =>
  `${count} ${count === 1 ? "event" : "events"}`;

// Importing is previewed first: nothing is saved until the user has seen
// what will be imported and what will be skipped.
export function IcsSettings({ tags, exportUrl, onImport }: IcsSettingsProps) {
  const [file, setFile] = useState<File | null>(null);
  const [tagId, setTagId] = useState("");
  const [preview, setPreview] = useState<IcsImportResult | null>(null);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  const run = async (dryRun: boolean) => {
    if (!file) return;
    if (file.size > maxFileBytes) {
      setMessage("The file is too large. The limit is 5 MB.");
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      const result = await onImport({ ics: await file.text(), tagId, dryRun });
      if (dryRun) {
        setPreview(result);
      } else {
        setPreview(null);
        setMessage(`Imported ${eventCount(result.imported)}.`);
      }
    } catch (error) {
      setMessage(getRequestErrorMessage(error, "Unable to import the file"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="settings-section" aria-labelledby="ics-heading">
      <h2 id="ics-heading">Import and export</h2>
      <p>
        Move events to and from other calendars with .ics files, including
        repeating ones. Tasks are not included yet.
      </p>
      <a className="primary-button compact ics-export" href={exportUrl}>
        Download my events (.ics)
      </a>

      <label className="settings-select-label" htmlFor="ics-file">
        Import a calendar file
      </label>
      <input
        id="ics-file"
        type="file"
        accept=".ics,text/calendar"
        onChange={(event) => {
          setFile(event.target.files?.[0] ?? null);
          setPreview(null);
          setMessage("");
        }}
      />
      <label className="settings-select-label" htmlFor="ics-tag">
        Tag imported events with
      </label>
      <select
        id="ics-tag"
        value={tagId}
        onChange={(event) => {
          setTagId(event.target.value);
          setPreview(null);
        }}
      >
        <option value="">Default tag</option>
        {tags.map((tag) => (
          <option key={tag.id} value={tag.id}>
            {tag.name}
          </option>
        ))}
      </select>
      <div className="ics-actions">
        <button
          className="primary-button compact"
          type="button"
          disabled={!file || busy || preview?.imported === 0}
          onClick={() => void run(!preview)}
        >
          {preview ? `Import ${eventCount(preview.imported)}` : "Preview"}
        </button>
      </div>

      {preview && (
        <div className="ics-preview" role="status">
          <p className="form-hint">
            {eventCount(preview.imported)} will be imported
            {preview.skipped.length > 0 &&
              `, ${preview.skipped.length} skipped`}
            .
          </p>
          {preview.skipped.length > 0 && (
            <ul aria-label="Skipped events">
              {preview.skipped.map((skipped, index) => (
                <li key={index}>
                  {skipped.title}: {skipped.reason}
                </li>
              ))}
            </ul>
          )}
          {preview.warnings.length > 0 && (
            <ul aria-label="Warnings">
              {preview.warnings.map((warning, index) => (
                <li key={index}>{warning}</li>
              ))}
            </ul>
          )}
        </div>
      )}
      {message && (
        <p className="form-hint" role="status">
          {message}
        </p>
      )}
    </section>
  );
}
