import { useState, useCallback, useRef, useEffect } from 'react';
import {
  FolderOpen,
  File,
  FileSymlink,
  ChevronRight,
  Upload,
  Download,
  ArrowUp,
  Loader2,
  HardDrive,
  Home,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { toast } from '@/components/ui/sonner';
import {
  listContainerFiles,
  getContainerFileDownloadUrl,
  uploadContainerFile,
  type ContainerFileEntry,
} from '@/services/backendApiClient';
import { useClusterStore } from '@/stores/clusterStore';

export interface FileTransferDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  podName: string;
  namespace: string;
  baseUrl?: string;
  clusterId?: string;
  containers?: Array<{ name: string }>;
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
      return <FolderOpen className="h-4 w-4 text-blue-400" />;
    case 'link':
      return <FileSymlink className="h-4 w-4 text-purple-400" />;
    default:
      return <File className="h-4 w-4 text-muted-foreground" />;
  }
}

export function FileTransferDialog({
  open,
  onOpenChange,
  podName,
  namespace,
  baseUrl,
  clusterId,
  containers,
}: FileTransferDialogProps) {
  const { backendUrl } = useClusterStore();
  const effectiveBaseUrl = baseUrl || backendUrl;
  const effectiveClusterId = clusterId || useClusterStore.getState().activeCluster?.id;

  const [currentPath, setCurrentPath] = useState('/');
  const [entries, setEntries] = useState<ContainerFileEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedContainer, setSelectedContainer] = useState(containers?.[0]?.name || '');
  const [uploading, setUploading] = useState(false);
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadDirectory = useCallback(
    async (dirPath: string) => {
      if (!effectiveBaseUrl || !effectiveClusterId) {
        setError('Missing backend URL or cluster ID');
        return;
      }
      setLoading(true);
      setError(null);
      try {
        const result = await listContainerFiles(
          effectiveBaseUrl,
          effectiveClusterId,
          namespace,
          podName,
          dirPath,
          selectedContainer
        );
        setEntries(result || []);
        setCurrentPath(dirPath);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to list files');
        setEntries([]);
      } finally {
        setLoading(false);
      }
    },
    [effectiveBaseUrl, effectiveClusterId, namespace, podName, selectedContainer]
  );

  // Load root directory when dialog opens
  useEffect(() => {
    if (open && selectedContainer) {
      loadDirectory('/');
    }
  }, [open, selectedContainer, loadDirectory]);

  // Update selected container when containers prop changes
  useEffect(() => {
    if (containers?.length && !selectedContainer) {
      setSelectedContainer(containers[0].name);
    }
  }, [containers, selectedContainer]);

  const navigateTo = (entry: ContainerFileEntry) => {
    if (entry.type === 'dir') {
      const newPath = currentPath === '/' ? `/${entry.name}` : `${currentPath}/${entry.name}`;
      loadDirectory(newPath);
    }
  };

  const navigateUp = () => {
    if (currentPath === '/') return;
    const parts = currentPath.split('/').filter(Boolean);
    parts.pop();
    loadDirectory('/' + parts.join('/') || '/');
  };

  const navigateToBreadcrumb = (index: number) => {
    if (index === -1) {
      loadDirectory('/');
      return;
    }
    const parts = currentPath.split('/').filter(Boolean);
    loadDirectory('/' + parts.slice(0, index + 1).join('/'));
  };

  const handleDownload = (entry: ContainerFileEntry) => {
    if (!effectiveBaseUrl || !effectiveClusterId) return;
    const filePath = currentPath === '/' ? `/${entry.name}` : `${currentPath}/${entry.name}`;
    const url = getContainerFileDownloadUrl(
      effectiveBaseUrl,
      effectiveClusterId,
      namespace,
      podName,
      filePath,
      selectedContainer
    );
    // Open in new tab to trigger browser download
    window.open(url, '_blank');
    toast.success(`Downloading ${entry.name}`);
  };

  const handleUploadFiles = async (files: FileList | File[]) => {
    if (!effectiveBaseUrl || !effectiveClusterId) return;
    setUploading(true);
    let successCount = 0;
    let failCount = 0;

    for (const file of Array.from(files)) {
      const destPath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      try {
        await uploadContainerFile(
          effectiveBaseUrl,
          effectiveClusterId,
          namespace,
          podName,
          destPath,
          selectedContainer,
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

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    if (e.dataTransfer.files.length > 0) {
      handleUploadFiles(e.dataTransfer.files);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => {
    setIsDragOver(false);
  };

  const pathParts = currentPath.split('/').filter(Boolean);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HardDrive className="h-5 w-5" />
            File Browser — {podName}
          </DialogTitle>
          <DialogDescription>
            Browse, download, and upload files in the container filesystem
          </DialogDescription>
        </DialogHeader>

        {/* Container selector */}
        {containers && containers.length > 1 && (
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">Container:</span>
            <div className="flex gap-1">
              {containers.map((c) => (
                <Badge
                  key={c.name}
                  variant={c.name === selectedContainer ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setSelectedContainer(c.name)}
                >
                  {c.name}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Breadcrumb navigation */}
        <div className="flex items-center gap-1 text-sm border rounded-md px-3 py-2 bg-muted/50 overflow-x-auto">
          <button
            onClick={() => navigateToBreadcrumb(-1)}
            className="flex items-center gap-1 hover:text-foreground text-muted-foreground transition-colors"
          >
            <Home className="h-3.5 w-3.5" />
            <span>/</span>
          </button>
          {pathParts.map((part, i) => (
            <span key={i} className="flex items-center gap-1">
              <ChevronRight className="h-3 w-3 text-muted-foreground" />
              <button
                onClick={() => navigateToBreadcrumb(i)}
                className={`hover:text-foreground transition-colors ${
                  i === pathParts.length - 1 ? 'text-foreground font-medium' : 'text-muted-foreground'
                }`}
              >
                {part}
              </button>
            </span>
          ))}
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={navigateUp}
            disabled={currentPath === '/' || loading}
          >
            <ArrowUp className="h-4 w-4 mr-1" />
            Up
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => loadDirectory(currentPath)}
            disabled={loading}
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Refresh'}
          </Button>
          <div className="flex-1" />
          <Button
            variant="outline"
            size="sm"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading || !effectiveBaseUrl || !effectiveClusterId}
          >
            <Upload className="h-4 w-4 mr-1" />
            {uploading ? 'Uploading...' : 'Upload'}
          </Button>
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

        {/* File list / Drop zone */}
        <div
          className={`flex-1 min-h-0 border rounded-md transition-colors ${
            isDragOver ? 'border-primary bg-primary/5' : ''
          }`}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
        >
          {error ? (
            <div className="flex items-center justify-center h-48 text-destructive text-sm">
              {error}
            </div>
          ) : loading && entries.length === 0 ? (
            <div className="flex items-center justify-center h-48 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" />
              Loading...
            </div>
          ) : entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-48 text-muted-foreground text-sm gap-2">
              <Upload className="h-8 w-8 opacity-50" />
              <span>Empty directory — drop files here to upload</span>
            </div>
          ) : (
            <ScrollArea className="h-[360px]">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-background border-b">
                  <tr className="text-muted-foreground text-xs">
                    <th className="text-left py-2 px-3 font-medium">Name</th>
                    <th className="text-right py-2 px-3 font-medium w-20">Size</th>
                    <th className="text-right py-2 px-3 font-medium w-40">Modified</th>
                    <th className="text-right py-2 px-3 font-medium w-16"></th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((entry) => (
                    <tr
                      key={entry.name}
                      className="hover:bg-muted/50 border-b border-border/50 transition-colors"
                    >
                      <td className="py-1.5 px-3">
                        <button
                          className={`flex items-center gap-2 ${
                            entry.type === 'dir' ? 'hover:text-blue-400 cursor-pointer' : ''
                          }`}
                          onClick={() => entry.type === 'dir' ? navigateTo(entry) : undefined}
                          disabled={entry.type !== 'dir'}
                        >
                          {fileIcon(entry.type)}
                          <span className="truncate max-w-[300px]">{entry.name}</span>
                          {entry.type === 'link' && (
                            <Badge variant="outline" className="text-[10px] px-1 py-0">
                              link
                            </Badge>
                          )}
                        </button>
                      </td>
                      <td className="py-1.5 px-3 text-right text-muted-foreground text-xs tabular-nums">
                        {entry.type === 'dir' ? '-' : formatSize(entry.size)}
                      </td>
                      <td className="py-1.5 px-3 text-right text-muted-foreground text-xs tabular-nums">
                        {entry.modified || '-'}
                      </td>
                      <td className="py-1.5 px-3 text-right">
                        {entry.type === 'file' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 w-6 p-0"
                            onClick={() => handleDownload(entry)}
                            title={`Download ${entry.name}`}
                          >
                            <Download className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </ScrollArea>
          )}
        </div>

        {/* Drop zone hint */}
        {isDragOver && (
          <div className="absolute inset-0 flex items-center justify-center bg-background/80 rounded-lg pointer-events-none z-50">
            <div className="flex flex-col items-center gap-2 text-primary">
              <Upload className="h-10 w-10" />
              <span className="text-lg font-medium">Drop files to upload to {currentPath}</span>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
