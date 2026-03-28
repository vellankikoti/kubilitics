import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface KubernetesConfig {
  apiUrl: string;
  token?: string;
  isConnected: boolean;
  lastConnected?: string;
}

interface KubernetesConfigStore {
  config: KubernetesConfig;
  setApiUrl: (url: string) => void;
  setToken: (token: string) => void;
  setConnected: (connected: boolean) => void;
  disconnect: () => void;
}

export const useKubernetesConfigStore = create<KubernetesConfigStore>()(
  persist(
    (set) => ({
      config: {
        apiUrl: '',
        token: undefined,
        isConnected: false,
      },
      setApiUrl: (url) => set((state) => ({ config: { ...state.config, apiUrl: url } })),
      setToken: (token) => set((state) => ({ config: { ...state.config, token } })),
      setConnected: (connected) =>
        set((state) => ({
          config: {
            ...state.config,
            isConnected: connected,
            lastConnected: connected ? new Date().toISOString() : state.config.lastConnected,
          },
        })),
      disconnect: () =>
        set({
          config: {
            apiUrl: '',
            token: undefined,
            isConnected: false,
          },
        }),
    }),
    {
      name: 'kubernetes-config',
      // Version 1: strip token from persistence.
      // The migrate function clears stale tokens from users who ran version 0.
      version: 1,
      migrate: (persisted: unknown) => {
        const state = persisted as Record<string, unknown> | null;
        if (state && typeof state === 'object') {
          const config = (state as { config?: Record<string, unknown> }).config;
          if (config && 'token' in config) {
            delete config.token;
          }
        }
        return state as ReturnType<typeof Object>;
      },
      partialize: (state) => ({
        config: {
          apiUrl: state.config.apiUrl,
          isConnected: state.config.isConnected,
          lastConnected: state.config.lastConnected,
          // token is intentionally excluded from persistence — never store credentials in localStorage
        },
      }),
    }
  )
);
