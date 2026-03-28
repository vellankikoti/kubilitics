import { useEffect, useRef, useMemo } from 'react';

// ---------------------------------------------------------------------------
// Platform detection
// ---------------------------------------------------------------------------

function isMac(): boolean {
  return /Mac|iPod|iPhone|iPad/.test(navigator.userAgent);
}

/** Returns the platform-appropriate modifier symbol. */
function modKey(): string {
  return isMac() ? '\u2318' : 'Ctrl';
}

// ---------------------------------------------------------------------------
// Shortcut data
// ---------------------------------------------------------------------------

interface ShortcutEntry {
  keys: string;
  description: string;
}

interface ShortcutSection {
  title: string;
  shortcuts: ShortcutEntry[];
}

function getSections(): ShortcutSection[] {
  const mod = modKey();
  return [
    {
      title: 'Navigation',
      shortcuts: [
        { keys: `${mod}+K`, description: 'Open search' },
        { keys: 'G then D', description: 'Go to Dashboard' },
        { keys: 'G then T', description: 'Go to Topology' },
        { keys: 'G then P', description: 'Go to Pods' },
        { keys: 'G then N', description: 'Go to Nodes' },
        { keys: 'G then S', description: 'Go to Settings' },
        { keys: '/', description: 'Focus search' },
      ],
    },
    {
      title: 'Topology',
      shortcuts: [
        { keys: 'F', description: 'Fit to screen' },
        { keys: '+  /  \u2212', description: 'Zoom in / out' },
        { keys: 'E', description: 'Toggle edge labels' },
        { keys: 'M', description: 'Toggle minimap' },
        { keys: 'H', description: 'Toggle health overlay' },
        { keys: 'S', description: 'Screenshot to clipboard' },
        { keys: '1\u20133', description: 'Switch view mode' },
        { keys: 'Backspace', description: 'Navigate back' },
      ],
    },
    {
      title: 'General',
      shortcuts: [
        { keys: '?', description: 'Show this dialog' },
        { keys: 'Esc', description: 'Close dialog / panel' },
        { keys: `${mod}+B`, description: 'Toggle sidebar' },
      ],
    },
  ];
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function KeyboardShortcutsOverlay({
  visible,
  onClose,
}: {
  visible: boolean;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const sections = useMemo(getSections, []);

  // Auto-focus close button when visible
  useEffect(() => {
    if (visible) {
      closeRef.current?.focus();
    }
  }, [visible]);

  // Escape to close + focus trap
  useEffect(() => {
    if (!visible) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key === 'Tab') {
        const dialog = dialogRef.current;
        if (!dialog) return;
        const focusable = dialog.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        if (focusable.length === 0) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown, { capture: true });
    return () => window.removeEventListener('keydown', handleKeyDown, { capture: true });
  }, [visible, onClose]);

  if (!visible) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
      onClick={onClose}
      role="presentation"
    >
      <div
        ref={dialogRef}
        className="w-[520px] max-h-[80vh] overflow-y-auto rounded-xl border border-slate-200/60 dark:border-slate-700/50 bg-background p-6 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label="Keyboard shortcuts"
      >
        {/* Header */}
        <div className="mb-5 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-slate-100 dark:bg-slate-800">
              <svg
                className="w-4 h-4 text-slate-500 dark:text-slate-400"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z"
                />
              </svg>
            </div>
            <h2 className="text-base font-semibold text-foreground">Keyboard Shortcuts</h2>
          </div>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 transition-colors"
            aria-label="Close shortcuts dialog"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Sections */}
        <div className="space-y-5">
          {sections.map((section) => (
            <div key={section.title}>
              <h3 className="text-[11px] font-bold uppercase tracking-widest text-muted-foreground mb-2.5">
                {section.title}
              </h3>
              <div className="space-y-0.5">
                {section.shortcuts.map((s) => (
                  <div
                    key={s.keys}
                    className="flex items-center justify-between py-1.5 text-sm"
                  >
                    <span className="text-muted-foreground">{s.description}</span>
                    <div className="flex items-center gap-1">
                      {s.keys.split(/ then | \+ /).map((part, i) => (
                        <span key={i} className="flex items-center gap-1">
                          {i > 0 && s.keys.includes('then') && (
                            <span className="text-[10px] text-muted-foreground/60 mx-0.5">then</span>
                          )}
                          {i > 0 && s.keys.includes('+') && (
                            <span className="text-[10px] text-muted-foreground/60 mx-0.5">+</span>
                          )}
                          <kbd className="inline-flex items-center justify-center min-w-[24px] rounded-md bg-muted px-2 py-1 font-mono text-xs font-medium text-foreground border border-slate-200 dark:border-slate-700 shadow-sm">
                            {part.trim()}
                          </kbd>
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800 text-center">
          <p className="text-[11px] text-muted-foreground">
            Press{' '}
            <kbd className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono border border-slate-200 dark:border-slate-700">
              Esc
            </kbd>{' '}
            to close
          </p>
        </div>
      </div>
    </div>
  );
}
