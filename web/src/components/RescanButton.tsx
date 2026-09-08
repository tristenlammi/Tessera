// RescanButton is the small ↻ that sits in the corner of a card whose figures come from
// a cached library pass rather than a live walk. Its tooltip says how old the figures
// are; pressing it starts a fresh pass. Deliberately quiet — the point of caching is that
// nobody needs to press this.
export function RescanButton({ busy, onClick, title }: { busy: boolean; onClick: () => void; title: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      title={title}
      aria-label={title}
      className={`rounded-md px-1 text-[14px] leading-none opacity-50 hover:opacity-100 disabled:opacity-40 ${busy ? "animate-spin" : ""}`}
      style={{ color: "var(--ink-dim)" }}
    >
      ↻
    </button>
  );
}

// ago renders a unix timestamp as "just now" / "12 min ago" / "3 h ago".
export function ago(ts: number): string {
  const s = Math.max(0, Math.floor(Date.now() / 1000 - ts));
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)} min ago`;
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`;
  return `${Math.floor(s / 86400)} d ago`;
}

// scanTitle is the tooltip for a card backed by a scan that reports when it last ran.
export function scanTitle(c: { scanned_at: number; scanning: boolean } | null): string {
  if (!c) return "Rescan the library";
  if (c.scanning) return "Scanning the library…";
  return c.scanned_at ? `Rescan the library · last scanned ${ago(c.scanned_at)}` : "Rescan the library";
}
