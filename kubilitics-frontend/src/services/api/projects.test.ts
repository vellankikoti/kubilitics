/**
 * Tests for src/services/api/projects.ts
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
  getProjects,
  getProject,
  createProject,
  updateProject,
  deleteProject,
  addClusterToProject,
  removeClusterFromProject,
  addNamespaceToProject,
  removeNamespaceFromProject,
} from './projects';
import { resetBackendCircuit, markBackendReady } from './client';

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

describe('getProjects (listProjects)', () => {
  it('returns array of projects', async () => {
    const projects = [
      { id: 'p1', name: 'Frontend', description: 'Frontend services' },
      { id: 'p2', name: 'Backend', description: 'Backend services' },
    ];
    mockFetch(projects);

    const result = await getProjects('http://localhost:8190');
    expect(result).toEqual(projects);
    expect(result).toHaveLength(2);
  });

  it('returns empty array when no projects exist', async () => {
    mockFetch([]);

    const result = await getProjects('http://localhost:8190');
    expect(result).toEqual([]);
  });

  it('calls correct endpoint path', async () => {
    mockFetch([]);

    await getProjects('http://localhost:8190');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('/projects');
  });
});

describe('getProject', () => {
  it('returns project with details', async () => {
    const project = { id: 'p1', name: 'Frontend', description: 'FE', clusters: [], namespaces: [] };
    mockFetch(project);

    const result = await getProject('http://localhost:8190', 'p1');
    expect(result).toEqual(project);
  });

  it('encodes projectId in the URL', async () => {
    mockFetch({ id: 'p1', name: 'Test' });

    await getProject('http://localhost:8190', 'project/special');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('project%2Fspecial');
  });
});

describe('createProject', () => {
  it('sends POST with name and description', async () => {
    const created = { id: 'p3', name: 'New Project', description: 'A new project' };
    mockFetch(created);

    const result = await createProject('http://localhost:8190', 'New Project', 'A new project');

    expect(result).toEqual(created);
    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ name: 'New Project', description: 'A new project' });
  });

  it('sends empty description when not provided', async () => {
    mockFetch({ id: 'p4', name: 'Minimal' });

    await createProject('http://localhost:8190', 'Minimal');

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(JSON.parse(init.body).description).toBe('');
  });
});

describe('updateProject', () => {
  it('sends PATCH with updated fields', async () => {
    mockFetch({ id: 'p1', name: 'Updated', description: 'Updated desc' });

    await updateProject('http://localhost:8190', 'p1', { name: 'Updated', description: 'Updated desc' });

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(init.method).toBe('PATCH');
    expect(JSON.parse(init.body)).toEqual({ name: 'Updated', description: 'Updated desc' });
  });

  it('sends partial update with only name', async () => {
    mockFetch({ id: 'p1', name: 'Renamed' });

    await updateProject('http://localhost:8190', 'p1', { name: 'Renamed' });

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ name: 'Renamed' });
  });
});

describe('deleteProject', () => {
  it('sends DELETE to correct endpoint', async () => {
    mockFetch('', 200);

    await deleteProject('http://localhost:8190', 'p1');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/projects/p1');
    expect(init.method).toBe('DELETE');
  });

  it('encodes projectId in the URL', async () => {
    mockFetch('', 200);

    await deleteProject('http://localhost:8190', 'project with spaces');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('project%20with%20spaces');
  });
});

describe('addClusterToProject', () => {
  it('sends POST with cluster_id', async () => {
    mockFetch('', 200);

    await addClusterToProject('http://localhost:8190', 'p1', 'cluster-abc');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/projects/p1/clusters');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ cluster_id: 'cluster-abc' });
  });
});

describe('removeClusterFromProject', () => {
  it('sends DELETE with correct path', async () => {
    mockFetch('', 200);

    await removeClusterFromProject('http://localhost:8190', 'p1', 'cluster-abc');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/projects/p1/clusters/cluster-abc');
    expect(init.method).toBe('DELETE');
  });
});

describe('addNamespaceToProject', () => {
  it('sends POST with namespace data', async () => {
    mockFetch('', 200);

    await addNamespaceToProject('http://localhost:8190', 'p1', 'cluster-1', 'default', 'platform');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/projects/p1/namespaces');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({
      cluster_id: 'cluster-1',
      namespace_name: 'default',
      team: 'platform',
    });
  });

  it('sends empty team when not provided', async () => {
    mockFetch('', 200);

    await addNamespaceToProject('http://localhost:8190', 'p1', 'cluster-1', 'kube-system');

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(JSON.parse(init.body).team).toBe('');
  });
});

describe('removeNamespaceFromProject', () => {
  it('sends DELETE with correct path', async () => {
    mockFetch('', 200);

    await removeNamespaceFromProject('http://localhost:8190', 'p1', 'cluster-1', 'default');

    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toContain('/projects/p1/namespaces/cluster-1/default');
    expect(init.method).toBe('DELETE');
  });
});
