import { useState, useCallback, useRef, useEffect } from 'react';
import {
  FolderOpen,
  File,
  FileSymlink,
  FileText,
  ChevronRight,
  ChevronDown,
  Upload,
  Download,
  Loader2,
  Home,
  RefreshCw,
  AlertCircle,
  FolderPlus,
  Eye,
  X,
  ArrowUp,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { toast } from '@/components/ui/sonner';
import { cn } from '@/lib/utils';
import {
  listContainerFiles,
  getContainerFileDownloadUrl,
  uploadContainerFile,
  type ContainerFileEntry,
} from '@/services/backendApiClient';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface PVCFileBrowserProps {
  podName: string;
  namespace: string;
  containerName: string;
  mountPath: string;
  baseUrl: string;
  clusterId: string;
}

interface TreeNode extends ContainerFileEntry {
  path: string;
  children?: TreeNode[];
  expanded?: boolean;
  loading?: boolean;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const TEXT_EXTENSIONS = new Set([
  'txt', 'md', 'json', 'yaml', 'yml', 'xml', 'csv', 'log', 'conf', 'cfg',
  'ini', 'sh', 'bash', 'zsh', 'py', 'js', 'ts', 'jsx', 'tsx', 'html',
  'css', 'scss', 'sql', 'env', 'toml', 'properties', 'gitignore',
  'dockerfile', 'makefile', 'rs', 'go', 'java', 'rb', 'php', 'c', 'cpp',
  'h', 'hpp',
]);

function isTextFile(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() ?? '';
  if (TEXT_EXTENSIONS.has(ext)) return true;
  // Common config files without extensions
  const lowerName = name.toLowerCase();
  return ['dockerfile', 'makefile', 'vagrantfile', 'gemfile', 'rakefile', '.gitignore', '.env'].some(
    (n) => lowerName === n || lowerName.endsWith(n)
  );
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, i);
  return `${value < 10 && i > 0 ? value.toFixed(1) : Math.round(value)} ${units[i]}`;
}

function fileIcon(type: string) {
  switch (type) {
    case 'dir':
      return <FolderOpen className="h-4 w-4 text-amber-500 dark:text-amber-400 shrink-0" />;
    case 'link':
      return <FileSymlink className="h-4 w-4 text-violet-500 dark:text-violet-400 shrink-0" />;
    default:
      return <File className="h-4 w-4 text-slate-400 dark:text-slate-500 shrink-0" />;
  }
}

function joinPath(base: string, name: string): string {
  if (base === '/') return `/${name}`;
  return `${base}/${name}`;
}

const MAX_PREVIEW_SIZE = 512 * 1024; // 512 KB

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function PVCFileBrowser({
  podName,
  namespace,
  containerName,
  mountPath,
  baseUrl,
  clusterId,
}: PVCFileBrowserProps) {
  const [currentPath, setCurrentPath] = useState(mountPath);
  const [entries, setEntries] = useState<ContainerFileEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Preview state
  const [previewFile, setPreviewFile] = useState<{ name: string; path: string } | null>(null);
  const [previewContent, setPreviewContent] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  // Tree state: track expanded directories
  const [expandedDirs, setExpandedDirs] = useState<Record<string, TreeNode[]>>({});
  const [expandedLoading, setExpandedLoading] = useState<Set<string>>(new Set());

  // ── Directory loading ────────────────────────────────────────────────

  const loadDirectory = useCallback(
    async (dirPath: string) => {
      setLoading(true);
      setError(null);
      try {
        const result = await listContainerFiles(
          baseUrl,
          clusterId,
          namespace,
          podName,
          dirPath,
          containerName
        );
        setEntries(result || []);
        setCurrentPath(dirPath);
        // Clear tree expansion when navigating via breadcrumb
        setExpandedDirs({});
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to list files');
        setEntries([]);
      } finally {
        setLoading(false);
      }
    },
    [baseUrl, clusterId, namespace, podName, containerName]
  );

  useEffect(() => {
    loadDirectory(mountPath);
  }, [mountPath, loadDirectory]);

  // ── Navigation ───────────────────────────────────────────────────────

  const navigateTo = (dirPath: string) => {
    loadDirectory(dirPath);
  };

  const navigateUp = () => {
    if (currentPath === mountPath || currentPath === '/') return;
    const parts = currentPath.split('/').filter(Boolean);
    parts.pop();
    const parent = '/' + parts.join('/') || '/';
    // Don't navigate above mountPath
    if (parent.length < mountPath.length && !mountPath.startsWith(parent + '/') && parent !== mountPath) {
      loadDirectory(mountPath);
    } else {
      loadDirectory(parent);
    }
  };

  const navigateToBreadcrumb = (index: number) => {
    const mountParts = mountPath.split('/').filter(Boolean);
    if (index < mountParts.length) {
      loadDirectory(mountPath);
      return;
    }
    const allParts = currentPath.split('/').filter(Boolean);
    loadDirectory('/' + allParts.slice(0, index + 1).join('/'));
  };

  // ── Tree expand/collapse ─────────────────────────────────────────────

  const toggleExpand = useCallback(
    async (dirPath: string) => {
      if (expandedDirs[dirPath]) {
        // Collapse
        setExpandedDirs((prev) => {
          const next = { ...prev };
          delete next[dirPath];
          return next;
        });
        return;
      }
      // Expand: load children
      setExpandedLoading((prev) => new Set(prev).add(dirPath));
      try {
        const children = await listContainerFiles(
          baseUrl,
          clusterId,
          namespace,
          podName,
          dirPath,
          containerName
        );
        const nodes: TreeNode[] = (children || []).map((c) => ({
          ...c,
          path: joinPath(dirPath, c.name),
        }));
        setExpandedDirs((prev) => ({ ...prev, [dirPath]: nodes }));
      } catch {
        toast.error(`Failed to list ${dirPath}`);
      } finally {
        setExpandedLoading((prev) => {
          const next = new Set(prev);
          next.delete(dirPath);
          return next;
        });
      }
    },
    [baseUrl, clusterId, namespace, podName, containerName, expandedDirs]
  );

  // ── File preview ─────────────────────────────────────────────────────

  const openPreview = useCallback(
    async (name: string, filePath: string) => {
      setPreviewFile({ name, path: filePath });
      setPreviewContent(null);
      setPreviewError(null);

      if (!isTextFile(name)) {
        // Not a text file, just show download button
        return;
      }

      setPreviewLoading(true);
      try {
        const url = getContainerFileDownloadUrl(
          baseUrl,
          clusterId,
          namespace,
          podName,
          filePath,
          containerName
        );
        const resp = await fetch(url);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);

        const contentLength = resp.headers.get('content-length');
        if (contentLength && parseInt(contentLength) > MAX_PREVIEW_SIZE) {
          setPreviewError('File too large for preview. Use the download button instead.');
          return;
        }

        const text = await resp.text();
        if (text.length > MAX_PREVIEW_SIZE) {
          setPreviewContent(text.slice(0, MAX_PREVIEW_SIZE) + '\n\n--- Truncated (file too large) ---');
        } else {
          setPreviewContent(text);
        }
      } catch (e) {
        setPreviewError(e instanceof Error ? e.message : 'Failed to load preview');
      } finally {
        setPreviewLoading(false);
      }
    },
    [baseUrl, clusterId, namespace, podName, containerName]
  );

  const closePreview = () => {
    setPreviewFile(null);
    setPreviewContent(null);
    setPreviewError(null);
  };

  // ── Download ─────────────────────────────────────────────────────────

  const handleDownload = (name: string, filePath: string) => {
    const url = getContainerFileDownloadUrl(
      baseUrl,
      clusterId,
      namespace,
      podName,
      filePath,
      containerName
    );
    window.open(url, '_blank');
    toast.success(`Downloading ${name}`);
  };

  // ── Upload ───────────────────────────────────────────────────────────

  const handleUploadFiles = async (files: FileList | File[]) => {
    setUploading(true);
    let successCount = 0;
    let failCount = 0;

    for (const file of Array.from(files)) {
      const destPath = joinPath(currentPath, file.name);
      try {
        await uploadContainerFile(
          baseUrl,
          clusterId,
          namespace,
          podName,
          destPath,
          containerName,
          file
        );
        successCount++;
      } catch (e) {
        failCount++;
        toast.error(`Failed to upload ${file.name}: ${e instanceof Error ? e.message : 'Unknown error'}`);
      }
    }

    setUploading(false);
    if (successCount > 0) {
      toast.success(`Uploaded ${successCount} file${successCount > 1 ? 's' : ''}`);
      loadDirectory(currentPath);
    }
    if (failCount > 0 && successCount === 0) {
      toast.error(`All ${failCount} uploads failed`);
    }
  };

  // ── Derived data ─────────────────────────────────────────────────────

  const pathParts = currentPath.split('/').filter(Boolean);
  const dirs = entries.filter((e) => e.type === 'dir').sort((a, b) => a.name.localeCompare(b.name));
  const files = entries.filter((e) => e.type !== 'dir').sort((a, b) => a.name.localeCompare(b.name));
  const sortedEntries = [...dirs, ...files];
  const canGoUp = currentPath !== mountPath && currentPath !== '/';

  // ── Recursive tree row renderer ──────────────────────────────────────

  const renderTreeRow = (entry: ContainerFileEntry, parentPath: string, depth: number) => {
    const entryPath = joinPath(parentPath, entry.name);
    const isDir = entry.type === 'dir';
    const isExpanded = !!expandedDirs[entryPath];
    const isLoadingDir = expandedLoading.has(entryPath);
    const children = expandedDirs[entryPath];

    return (
      <div key={entryPath}>
        <div
          className={cn(
            'grid grid-cols-[1fr_5rem_9rem_4.5rem] items-center px-4 py-1.5 transition-colors group',
            isDir
              ? 'hover:bg-amber-50/50 dark:hover:bg-amber-900/10 cursor-pointer'
              : 'hover:bg-slate-50 dark:hover:bg-slate-800/40'
          )}
          onClick={() => {
            if (isDir) toggleExpand(entryPath);
          }}
          role={isDir ? 'button' : undefined}
        >
          {/* Name with indent */}
          <div className="flex items-center gap-1.5 min-w-0" style={{ paddingLeft: `${depth * 16}px` }}>
            {isDir ? (
              isLoadingDir ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground shrink-0" />
              ) : isExpanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              )
            ) : (
              <span className="w-3.5 shrink-0" />
            )}
            {fileIcon(entry.type)}
            <span
              className={cn(
                'text-sm truncate',
                isDir
                  ? 'font-medium text-slate-800 dark:text-slate-200 group-hover:text-amber-700 dark:group-hover:text-amber-400'
                  : 'text-slate-700 dark:text-slate-300'
              )}
            >
              {entry.name}
            </span>
            {entry.type === 'link' && (
              <Badge
                variant="outline"
                className="text-[10px] px-1 py-0 font-medium text-violet-500 border-violet-200 dark:border-violet-800"
              >
                symlink
              </Badge>
            )}
          </div>

          {/* Size */}
          <span className="text-right text-xs tabular-nums text-slate-400 dark:text-slate-500">
            {isDir ? <span className="text-[10px]">--</span> : formatSize(entry.size)}
          </span>

          {/* Modified */}
          <span className="text-right text-xs tabular-nums text-slate-400 dark:text-slate-500">
            {entry.modified || '--'}
          </span>

          {/* Actions */}
          <div className="flex justify-end gap-0.5">
            {entry.type === 'file' && (
              <>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    openPreview(entry.name, entryPath);
                  }}
                  className="h-6 w-6 rounded-md flex items-center justify-center text-slate-400 opacity-0 group-hover:opacity-100 hover:text-primary hover:bg-primary/10 transition-all"
                  title={`Preview ${entry.name}`}
                >
                  <Eye className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDownload(entry.name, entryPath);
                  }}
                  className="h-6 w-6 rounded-md flex items-center justify-center text-slate-400 opacity-0 group-hover:opacity-100 hover:text-primary hover:bg-primary/10 transition-all"
                  title={`Download ${entry.name}`}
                >
                  <Download className="h-3.5 w-3.5" />
                </button>
              </>
            )}
          </div>
        </div>

        {/* Expanded children */}
        {isExpanded && children && (
          <div>
            {[...children]
              .sort((a, b) => {
                if (a.type === 'dir' && b.type !== 'dir') return -1;
                if (a.type !== 'dir' && b.type === 'dir') return 1;
                return a.name.localeCompare(b.name);
              })
              .map((child) => renderTreeRow(child, entryPath, depth + 1))}
            {children.length === 0 && (
              <div
                className="text-xs text-muted-foreground italic px-4 py-1"
                style={{ paddingLeft: `${(depth + 1) * 16 + 16}px` }}
              >
                Empty directory
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  // ── Preview panel ────────────────────────────────────────────────────

  if (previewFile) {
    const isText = isTextFile(previewFile.name);
    return (
      <div className="flex flex-col h-full">
        {/* Preview header */}
        <div className="flex items-center gap-2 px-4 py-3 border-b border-slate-200/60 dark:border-slate-700/60 bg-slate-50/80 dark:bg-slate-800/80">
          <FileText className="h-4 w-4 text-primary shrink-0" />
          <span className="text-sm font-medium truncate flex-1">{previewFile.name}</span>
          <Button
            variant="outline"
            size="sm"
            className="h-7 gap-1 text-xs"
            onClick={() => handleDownload(previewFile.name, previewFile.path)}
          >
            <Download className="h-3.5 w-3.5" />
            Download
          </Button>
          <button
            onClick={closePreview}
            className="h-7 w-7 rounded-md flex items-center justify-center text-slate-500 hover:text-slate-900 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Preview body */}
        <div className="flex-1 min-h-0">
          {previewLoading ? (
            <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
              <Loader2 className="h-6 w-6 animate-spin text-primary/60" />
              <span className="text-sm">Loading preview...</span>
            </div>
          ) : previewError ? (
            <div className="flex flex-col items-center justify-center h-48 gap-2">
              <AlertCircle className="h-5 w-5 text-red-500" />
              <p className="text-sm text-red-600 dark:text-red-400">{previewError}</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDownload(previewFile.name, previewFile.path)}
              >
                <Download className="h-3.5 w-3.5 mr-1" />
                Download instead
              </Button>
            </div>
          ) : isText && previewContent != null ? (
            <ScrollArea className="h-[400px]">
              <pre className="p-4 text-xs font-mono text-slate-700 dark:text-slate-300 whitespace-pre-wrap break-all leading-relaxed">
                {previewContent}
              </pre>
            </ScrollArea>
          ) : (
            <div className="flex flex-col items-center justify-center h-48 gap-3 text-muted-foreground">
              <File className="h-8 w-8 text-slate-400" />
              <p className="text-sm">Binary file &mdash; preview not available</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDownload(previewFile.name, previewFile.path)}
              >
                <Download className="h-3.5 w-3.5 mr-1" />
                Download file
              </Button>
            </div>
          )}
        </div>
      </div>
    );
  }

  // ── Main file browser view ───────────────────────────────────────────

  return (
    <div className="flex flex-col h-full">
      {/* Breadcrumb + toolbar */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-slate-200/60 dark:border-slate-700/60 bg-slate-50/80 dark:bg-slate-800/80">
        {/* Breadcrumb */}
        <div className="flex-1 min-w-0 flex items-center gap-0.5 text-xs bg-white dark:bg-slate-800/80 border border-slate-200/80 dark:border-slate-700/60 rounded-lg px-2 py-1 overflow-x-auto">
          <button
            onClick={() => navigateTo(mountPath)}
            className="flex items-center gap-0.5 hover:text-primary text-muted-foreground transition-colors shrink-0"
          >
            <Home className="h-3 w-3" />
          </button>
          {pathParts.map((part, i) => (
            <span key={i} className="flex items-center gap-0.5 shrink-0">
              <ChevronRight className="h-3 w-3 text-slate-300 dark:text-slate-600" />
              <button
                onClick={() => navigateToBreadcrumb(i)}
                className={cn(
                  'hover:text-primary transition-colors truncate max-w-[120px]',
                  i === pathParts.length - 1
                    ? 'text-slate-900 dark:text-slate-100 font-medium'
                    : 'text-muted-foreground'
                )}
              >
                {part}
              </button>
            </span>
          ))}
        </div>

        {/* Nav buttons */}
        <button
          onClick={navigateUp}
          disabled={!canGoUp || loading}
          className="h-7 w-7 rounded-lg border border-slate-200/80 dark:border-slate-700/60 bg-white dark:bg-slate-800/80 flex items-center justify-center text-slate-500 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-300 dark:hover:border-slate-600 disabled:opacity-40 disabled:pointer-events-none transition-all"
          title="Go up"
        >
          <ArrowUp className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={() => loadDirectory(currentPath)}
          disabled={loading}
          className="h-7 w-7 rounded-lg border border-slate-200/80 dark:border-slate-700/60 bg-white dark:bg-slate-800/80 flex items-center justify-center text-slate-500 hover:text-slate-900 dark:hover:text-slate-100 hover:border-slate-300 dark:hover:border-slate-600 disabled:opacity-40 disabled:pointer-events-none transition-all"
          title="Refresh"
        >
          <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
        </button>
        <button
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          className="h-7 rounded-lg border border-primary/30 bg-primary/5 dark:bg-primary/10 px-2 flex items-center gap-1 text-xs font-medium text-primary hover:bg-primary/10 dark:hover:bg-primary/20 hover:border-primary/50 disabled:opacity-40 disabled:pointer-events-none transition-all"
          title="Upload files"
        >
          {uploading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Upload className="h-3.5 w-3.5" />
          )}
          <span className="hidden sm:inline">{uploading ? 'Uploading...' : 'Upload'}</span>
        </button>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => {
            if (e.target.files?.length) {
              handleUploadFiles(e.target.files);
              e.target.value = '';
            }
          }}
        />
      </div>

      {/* File tree */}
      <div className="flex-1 min-h-0">
        {error ? (
          <div className="flex flex-col items-center justify-center h-48 gap-2">
            <div className="h-10 w-10 rounded-full bg-red-50 dark:bg-red-900/20 flex items-center justify-center">
              <AlertCircle className="h-5 w-5 text-red-500" />
            </div>
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            <Button variant="outline" size="sm" onClick={() => loadDirectory(currentPath)} className="mt-1">
              Try Again
            </Button>
          </div>
        ) : loading && entries.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
            <Loader2 className="h-6 w-6 animate-spin text-primary/60" />
            <span className="text-sm">Loading directory...</span>
          </div>
        ) : entries.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 gap-3 text-muted-foreground">
            <div className="h-12 w-12 rounded-xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
              <FolderPlus className="h-6 w-6 text-slate-400" />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Empty directory</p>
              <p className="text-xs text-muted-foreground mt-0.5">No files found at this path</p>
            </div>
          </div>
        ) : (
          <ScrollArea className="h-[420px]">
            {/* Column headers */}
            <div className="sticky top-0 z-10 grid grid-cols-[1fr_5rem_9rem_4.5rem] items-center px-4 py-1 text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 bg-slate-50/95 dark:bg-slate-900/95 backdrop-blur-sm border-b border-slate-200/60 dark:border-slate-700/40">
              <span>Name</span>
              <span className="text-right">Size</span>
              <span className="text-right">Modified</span>
              <span />
            </div>

            {/* Tree rows */}
            <div className="divide-y divide-slate-100 dark:divide-slate-800/60">
              {sortedEntries.map((entry) => renderTreeRow(entry, currentPath, 0))}
            </div>

            {/* Summary */}
            <div className="sticky bottom-0 px-4 py-2 text-[10px] text-slate-400 dark:text-slate-500 bg-white/95 dark:bg-slate-900/95 backdrop-blur-sm border-t border-slate-200/60 dark:border-slate-700/40">
              {dirs.length} folder{dirs.length !== 1 ? 's' : ''}, {files.length} file
              {files.length !== 1 ? 's' : ''}
            </div>
          </ScrollArea>
        )}
      </div>
    </div>
  );
}
