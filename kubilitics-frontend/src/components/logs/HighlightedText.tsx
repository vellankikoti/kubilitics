/**
 * HighlightedText — wraps regex matches in <mark>. Shared between the plain
 * log view, JSON tree view, and structured log view so search highlighting
 * behaves consistently everywhere instead of only existing in one of them.
 */
export function HighlightedText({
  text,
  regex,
  isCurrent,
}: {
  text: string;
  regex: RegExp | null;
  /** The single "current" match under keyboard navigation gets a brighter marker. */
  isCurrent?: boolean;
}) {
  if (!regex) return <>{text}</>;
  // Clone to avoid mutating shared regex state across cells
  const re = new RegExp(regex.source, regex.flags);
  const parts: { text: string; match: boolean }[] = [];
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) parts.push({ text: text.slice(last, m.index), match: false });
    parts.push({ text: m[0], match: true });
    last = m.index + m[0].length;
    if (m[0].length === 0) { re.lastIndex++; }
  }
  if (last < text.length) parts.push({ text: text.slice(last), match: false });

  // bg-orange-500/40 + text-white was fixed for a dark background — on
  // light mode's white page that's low-contrast orange-on-white with white
  // text on top of it, exactly the "hard to see" complaint. Dark text on a
  // solid, opaque highlight reads clearly in both themes.
  const markCls = isCurrent
    ? 'bg-orange-500 text-black rounded-sm not-italic font-semibold ring-1 ring-orange-300'
    : 'bg-orange-400/70 text-black rounded-sm not-italic font-medium dark:bg-orange-500/50 dark:text-white';

  return (
    <>
      {parts.map((p, i) =>
        p.match ? (
          <mark key={i} className={markCls}>
            {p.text}
          </mark>
        ) : (
          <span key={i}>{p.text}</span>
        )
      )}
    </>
  );
}
