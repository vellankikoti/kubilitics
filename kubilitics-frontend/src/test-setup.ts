/**
 * Vitest global setup — runs before each test file.
 * Patches missing browser APIs that jsdom does not provide.
 */

// Monaco-editor calls document.queryCommandSupported at module-eval time
if (typeof document !== 'undefined' && !document.queryCommandSupported) {
  document.queryCommandSupported = () => false;
}

// Radix ScrollArea depends on ResizeObserver
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}
