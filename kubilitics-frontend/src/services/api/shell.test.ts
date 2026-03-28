/**
 * Tests for src/services/api/shell.ts
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@/lib/tauri', () => ({ isTauri: () => false }));
vi.mock('@/stores/authStore', () => ({
  useAuthStore: { getState: () => ({ logout: vi.fn() }) },
}));
vi.mock('@/stores/clusterStore', () => ({
  useClusterStore: { getState: () => ({ activeCluster: null, kubeconfigContent: null }) },
}));

import {
  getPodLogsUrl,
  getPodExecWebSocketUrl,
  getKubectlShellStreamUrl,
  getKCLIShellStreamUrl,
  postShellCommand,
  postKCLIExec,
  getShellComplete,
  getShellStatus,
  getKCLITUIState,
  getKCLIComplete,
} from './shell';
import { API_PREFIX, resetBackendCircuit, markBackendReady } from './client';

function mockFetch(body: unknown, status = 200) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers(),
    text: () => Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response);
}

beforeEach(() => {
  resetBackendCircuit();
  markBackendReady();
  vi.restoreAllMocks();
});

// ── URL generators (synchronous, no fetch) ──────────────────────────────────

describe('getPodLogsUrl', () => {
  it('generates correct URL without params', () => {
    const url = getPodLogsUrl('http://localhost:8190', 'c1', 'default', 'web-1');
    expect(url).toBe(`http://localhost:8190${API_PREFIX}/clusters/c1/logs/default/web-1`);
  });

  it('generates URL with container, tail, and follow params', () => {
    const url = getPodLogsUrl('http://localhost:8190', 'c1', 'default', 'web-1', {
      container: 'nginx',
      tail: 100,
      follow: true,
    });
    expect(url).toContain('container=nginx');
    expect(url).toContain('tail=100');
    expect(url).toContain('follow=true');
  });

  it('strips trailing slash from baseUrl', () => {
    const url = getPodLogsUrl('http://localhost:8190/', 'c1', 'default', 'web-1');
    expect(url).toMatch(/^http:\/\/localhost:8190\/api\//);
    expect(url).not.toContain('//api');
  });

  it('encodes namespace and pod name', () => {
    const url = getPodLogsUrl('http://localhost:8190', 'c1', 'my ns', 'pod/1');
    expect(url).toContain('my%20ns');
    expect(url).toContain('pod%2F1');
  });
});

describe('getPodExecWebSocketUrl', () => {
  it('generates ws:// URL from http:// baseUrl', () => {
    const url = getPodExecWebSocketUrl('http://localhost:8190', 'c1', 'default', 'web-1');
    expect(url).toMatch(/^ws:\/\//);
    expect(url).toContain(`${API_PREFIX}/clusters/c1/pods/default/web-1/exec`);
  });

  it('generates wss:// URL from https:// baseUrl', () => {
    const url = getPodExecWebSocketUrl('https://kubilitics.io', 'c1', 'default', 'web-1');
    expect(url).toMatch(/^wss:\/\//);
  });

  it('includes container and shell params', () => {
    const url = getPodExecWebSocketUrl('http://localhost:8190', 'c1', 'default', 'web-1', {
      container: 'app',
      shell: '/bin/bash',
    });
    expect(url).toContain('container=app');
    expect(url).toContain('shell=%2Fbin%2Fbash');
  });

  it('omits query string when no params', () => {
    const url = getPodExecWebSocketUrl('http://localhost:8190', 'c1', 'default', 'web-1');
    expect(url).not.toContain('?');
  });

  it('encodes special characters in namespace and pod name', () => {
    const url = getPodExecWebSocketUrl('http://localhost:8190', 'c1', 'my ns', 'pod/1');
    expect(url).toContain('my%20ns');
    expect(url).toContain('pod%2F1');
  });
});

describe('getKubectlShellStreamUrl', () => {
  it('generates ws:// URL with /shell/stream path', () => {
    const url = getKubectlShellStreamUrl('http://localhost:8190', 'c1');
    expect(url).toBe(`ws://localhost:8190${API_PREFIX}/clusters/c1/shell/stream`);
  });

  it('converts https to wss', () => {
    const url = getKubectlShellStreamUrl('https://kubilitics.io', 'c1');
    expect(url).toMatch(/^wss:\/\//);
  });

  it('strips trailing slashes from baseUrl', () => {
    const url = getKubectlShellStreamUrl('http://localhost:8190///', 'c1');
    expect(url).not.toContain('///');
  });
});

describe('getKCLIShellStreamUrl', () => {
  it('generates URL with mode=shell param', () => {
    const url = getKCLIShellStreamUrl('http://localhost:8190', 'c1');
    expect(url).toContain('mode=shell');
    expect(url).toContain('/kcli/stream');
  });

  it('includes namespace param when provided and not "all"', () => {
    const url = getKCLIShellStreamUrl('http://localhost:8190', 'c1', 'shell', 'kube-system');
    expect(url).toContain('namespace=kube-system');
  });

  it('omits namespace param when value is "all"', () => {
    const url = getKCLIShellStreamUrl('http://localhost:8190', 'c1', 'shell', 'all');
    expect(url).not.toContain('namespace=');
  });

  it('omits namespace param when undefined', () => {
    const url = getKCLIShellStreamUrl('http://localhost:8190', 'c1');
    expect(url).not.toContain('namespace=');
  });
});

// ── Async functions that call backendRequest ────────────────────────────────

describe('postShellCommand', () => {
  it('sends POST with command body', async () => {
    mockFetch({ output: 'NAME  READY  STATUS\nweb-1  1/1  Running', exit_code: 0 });

    const result = await postShellCommand('http://localhost:8190', 'c1', 'kubectl get pods');

    expect(result.output).toContain('web-1');
    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ command: 'kubectl get pods' });
  });

  it('trims whitespace from command', async () => {
    mockFetch({ output: '', exit_code: 0 });

    await postShellCommand('http://localhost:8190', 'c1', '  kubectl get ns  ');

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(JSON.parse(init.body).command).toBe('kubectl get ns');
  });
});

describe('postKCLIExec', () => {
  it('sends POST with args array', async () => {
    mockFetch({ output: 'done', exit_code: 0 });

    await postKCLIExec('http://localhost:8190', 'c1', ['get', 'pods']);

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ args: ['get', 'pods'], force: false });
  });

  it('sends X-Confirm-Destructive header when force=true', async () => {
    mockFetch({ output: 'deleted', exit_code: 0 });

    await postKCLIExec('http://localhost:8190', 'c1', ['delete', 'pod', 'web-1'], true);

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.headers['X-Confirm-Destructive']).toBe('true');
  });
});

describe('getShellComplete', () => {
  it('returns completions for a partial line', async () => {
    mockFetch({ completions: ['pods', 'pod-disruption-budgets'] });

    const result = await getShellComplete('http://localhost:8190', 'c1', 'kubectl get po');

    expect(result.completions).toContain('pods');
    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('line=kubectl%20get%20po');
  });

  it('omits query string for empty line', async () => {
    mockFetch({ completions: [] });

    await getShellComplete('http://localhost:8190', 'c1', '');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('?line=');
  });
});

describe('getShellStatus', () => {
  it('returns shell status', async () => {
    const status = { context: 'minikube', namespace: 'default', capabilities: ['exec', 'logs'] };
    mockFetch(status);

    const result = await getShellStatus('http://localhost:8190', 'c1');
    expect(result).toEqual(status);
  });
});

describe('getKCLITUIState', () => {
  it('returns TUI state', async () => {
    const state = { context: 'docker-desktop', namespace: 'default' };
    mockFetch(state);

    const result = await getKCLITUIState('http://localhost:8190', 'c1');
    expect(result).toEqual(state);
  });
});

describe('getKCLIComplete', () => {
  it('returns kcli completions', async () => {
    mockFetch({ completions: ['get', 'describe'] });

    const result = await getKCLIComplete('http://localhost:8190', 'c1', 'kcli ');

    expect(result.completions).toContain('get');
    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('/kcli/complete');
  });
});
