/**
 * Isolates the Vite-specific `new Worker(new URL(...))` construction
 * syntax in its own module so useElkLayout.ts's tests can mock this one
 * function instead of needing a real Worker environment (Vitest's jsdom
 * doesn't implement module workers).
 */
export function createElkLayoutWorker(): Worker {
  return new Worker(new URL("./elkLayout.worker.ts", import.meta.url), { type: "module" });
}
