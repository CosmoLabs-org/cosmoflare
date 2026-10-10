// Alert history as a real React view (P-06, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// views.ts's renderAlertList. Newest-first visible list (the store filters
// snoozed records), Acknowledge / Snooze 10m actions, the quiet empty
// state, and the severity classes (is-info / is-warning / is-critical).
// The store module (listAlerts, acknowledge, snooze — IndexedDB) and
// relativeTime are imported verbatim; the section masthead owns the page
// title.

import { useCallback, useEffect, useState } from "react";
import { acknowledge, listAlerts, snooze, type AlertRecord } from "../../store";
import { relativeTime } from "../../views";

/** Same mapping as views.ts's alertCard: unknown severities read as info. */
function severityClass(severity: string): string {
  return severity === "warning" || severity === "critical" ? severity : "info";
}

/** One alert card: severity mark + title, service/time meta, detail, and
 *  the Acknowledge / Snooze actions (both reload the list after writing). */
function AlertCard({ record, onChanged }: { record: AlertRecord; onChanged: () => void }): React.JSX.Element {
  const ack = (): void => {
    void acknowledge(record.id).then(onChanged);
  };
  const snz = (): void => {
    void snooze(record.id, 10).then(onChanged);
  };
  return (
    <article
      className={`cf-alert is-${severityClass(record.severity)}`}
      data-id={record.id}
      data-acked={record.acked ? "true" : undefined}
    >
      <div className="cf-alert-header">
        <span className="cf-severity-mark">{record.severity}</span>
        <strong className="cf-alert-title">{record.title}</strong>
      </div>
      <div className="cf-alert-meta">
        <span className="cf-alert-service">{record.service}</span>
        <time className="cf-alert-time">{relativeTime(record.received_at)}</time>
      </div>
      <p className="cf-alert-detail">{record.detail}</p>
      <div className="cf-alert-actions">
        <button type="button" className="cf-btn" onClick={ack}>
          {record.acked ? "Acknowledged" : "Acknowledge"}
        </button>
        <button type="button" className="cf-btn cf-btn-ghost" onClick={snz}>Snooze 10m</button>
      </div>
    </article>
  );
}

export default function AlertsView({ refreshSeq = 0 }: { refreshSeq?: number }): React.JSX.Element {
  const [records, setRecords] = useState<AlertRecord[] | null>(null);

  const load = useCallback((): void => {
    void listAlerts().then((visible) => setRecords(visible));
  }, []);

  // refreshSeq > 0 means the shell's global Refresh re-invoked the view.
  useEffect(() => {
    load();
  }, [load, refreshSeq]);

  if (records === null) {
    return (
      <div className="cf-loading">
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
      </div>
    );
  }

  if (records.length === 0) {
    return <p className="cf-empty">No alerts. Your daemon will page you here.</p>;
  }

  return (
    <div className="cf-alert-list">
      {records.map((record) => (
        <AlertCard key={record.id} record={record} onChanged={load} />
      ))}
    </div>
  );
}
