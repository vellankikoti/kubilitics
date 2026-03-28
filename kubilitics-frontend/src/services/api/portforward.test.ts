/**
 * Tests for src/services/api/portforward.ts
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
  startPortForward,
  stopPortForward,
  createDebugContainer,
  listContainerFiles,
  getContainerFileDownloadUrl,
} from './portforward';
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

describe('startPortForward', () => {
  it('returns session data on success', async () => {
    const session = { session_id: 'pf-123', local_port: 9090 };
    mockFetch(session);

    const result = await startPortForward('http://localhost:8190', 'c1', {
      namespace: 'default',
      pod: 'web-1',
      container_port: 8080,
      local_port: 9090,
    });

    expect(result).toEqual(session);
  });

  it('sends POST with correct body', async () => {
    mockFetch({ session_id: 'pf-1', local_port: 3000 });

    const req = { namespace: 'prod', pod: 'api-1', container_port: 80, local_port: 3000 };
    await startPortForward('http://localhost:8190', 'c1', req);

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/port-forward');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual(req);
  });
});

describe('stopPortForward', () => {
  it('sends DELETE to correct endpoint', async () => {
    mockFetch('', 200);

    await stopPortForward('http://localhost:8190', 'c1', 'pf-123');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/port-forward/pf-123');
    expect(init.method).toBe('DELETE');
  });

  it('encodes sessionId in the URL', async () => {
    mockFetch('', 200);

    await stopPortForward('http://localhost:8190', 'c1', 'session/with/slashes');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('session%2Fwith%2Fslashes');
  });
});

describe('createDebugContainer', () => {
  it('sends POST with image and target container', async () => {
    mockFetch({ name: 'debugger-abc', status: 'Running' });

    const result = await createDebugContainer(
      'http://localhost:8190', 'c1', 'default', 'web-1', 'busybox:latest', 'app'
    );

    expect(result).toEqual({ name: 'debugger-abc', status: 'Running' });
    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/debug');
    expect(init.method).toBe('POST');
    const body = JSON.parse(init.body);
    expect(body.image).toBe('busybox:latest');
    expect(body.targetContainer).toBe('app');
  });
});

describe('listContainerFiles', () => {
  it('returns file list from container', async () => {
    const files = [
      { name: 'app.js', size: 1024, isDir: false },
      { name: 'node_modules', size: 0, isDir: true },
    ];
    mockFetch(files);

    const result = await listContainerFiles(
      'http://localhost:8190', 'c1', 'default', 'web-1', '/app', 'nginx'
    );

    expect(result).toEqual(files);
    expect(result).toHaveLength(2);
  });

  it('sends POST with path and container in body', async () => {
    mockFetch([]);

    await listContainerFiles('http://localhost:8190', 'c1', 'ns', 'pod-1', '/var/log', 'sidecar');

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ path: '/var/log', container: 'sidecar' });
  });

  it('returns empty array for empty directory', async () => {
    mockFetch([]);

    const result = await listContainerFiles(
      'http://localhost:8190', 'c1', 'default', 'web-1', '/empty', 'app'
    );
    expect(result).toEqual([]);
  });
});

describe('getContainerFileDownloadUrl', () => {
  it('generates correct download URL', () => {
    const url = getContainerFileDownloadUrl(
      'http://localhost:8190', 'c1', 'default', 'web-1', '/app/config.yaml', 'nginx'
    );

    expect(url).toBe(
      `http://localhost:8190${API_PREFIX}/clusters/c1/resources/default/web-1/download?path=%2Fapp%2Fconfig.yaml&container=nginx`
    );
  });

  it('encodes special characters in path and container', () => {
    const url = getContainerFileDownloadUrl(
      'http://localhost:8190', 'c1', 'my ns', 'pod/1', '/path with spaces', 'my container'
    );

    expect(url).toContain('my%20ns');
    expect(url).toContain('pod%2F1');
    expect(url).toContain('path=%2Fpath%20with%20spaces');
    expect(url).toContain('container=my%20container');
  });

  it('strips trailing slash from baseUrl', () => {
    const url = getContainerFileDownloadUrl(
      'http://localhost:8190/', 'c1', 'default', 'web-1', '/file', 'app'
    );

    expect(url).toMatch(/^http:\/\/localhost:8190\/api\//);
    expect(url).not.toContain('//api');
  });
});
