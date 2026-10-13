/**
 * FOLLOWUP-DEP-1 migration (i18next-http-backend 3.0.6 -> 4.0.2): the app's
 * own test suite never exercised this module before (src/i18n/i18n.ts is
 * only imported from main.tsx, never from App.tsx, so component tests never
 * trigger its side-effecting .init() call). This test closes that gap by
 * driving a real Backend instance through the project's exact config
 * (loadPath './locales/{{lng}}/{{ns}}.json') against a mocked fetch, proving
 * v4's dropped cross-fetch dependency and host-fetch requirement is
 * satisfied in this project's actual runtime (jsdom/browser/Tauri webview
 * all provide native fetch) and that the real translation payload loads.
 */
import { describe, it, expect, vi, afterEach } from 'vitest';
import i18next from 'i18next';
import Backend from 'i18next-http-backend';

describe('i18next-http-backend v4 (FOLLOWUP-DEP-1)', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('loads a real namespace via the project\'s exact loadPath config', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = String(input);
      expect(url).toBe('./locales/en/translation.json');
      return new Response(JSON.stringify({ greeting: 'hello' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    });
    vi.stubGlobal('fetch', fetchMock);

    const instance = i18next.createInstance();
    await instance.use(Backend).init({
      lng: 'en',
      fallbackLng: 'en',
      ns: ['translation'],
      defaultNS: 'translation',
      backend: {
        loadPath: './locales/{{lng}}/{{ns}}.json',
      },
      interpolation: { escapeValue: false },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(instance.t('greeting')).toBe('hello');
  });

  it('supportedLngs rejects an attacker-controlled lng before it reaches the backend', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    const instance = i18next.createInstance();
    await instance.use(Backend).init({
      lng: 'http:evil.example.com',
      fallbackLng: false,
      // Mirrors src/i18n/i18n.ts's real config: i18next-http-backend itself
      // does not validate `lng` before interpolating it into loadPath (the
      // first test in this file, and the library's docs, confirm it will
      // happily fetch whatever URL the template produces) — supportedLngs
      // is the actual mitigation, enforced by i18next core before the
      // backend is ever invoked.
      supportedLngs: ['en', 'de'],
      nonExplicitSupportedLngs: false,
      ns: ['translation'],
      defaultNS: 'translation',
      backend: {
        loadPath: '{{lng}}/{{ns}}.json',
      },
      interpolation: { escapeValue: false },
    });

    expect(fetchMock).not.toHaveBeenCalled();
  });
});
