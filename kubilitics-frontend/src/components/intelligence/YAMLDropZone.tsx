import * as React from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { Upload, X, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface YAMLDropZoneProps {
  onAnalyze: (yaml: string) => void;
  onSetFilename: (name: string | null) => void;
  onClose: () => void;
  isLoading: boolean;
}

const YAML_PLACEHOLDER = `# Paste your Kubernetes manifest here, e.g.:
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-app
          image: nginx:1.25`;

export function YAMLDropZone({
  onAnalyze,
  onSetFilename,
  onClose,
  isLoading,
}: YAMLDropZoneProps) {
  const [yaml, setYaml] = React.useState('');
  const [isDragOver, setIsDragOver] = React.useState(false);
  const [filename, setFilename] = React.useState<string | null>(null);
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const readFile = React.useCallback(
    (file: File) => {
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = e.target?.result as string;
        setYaml(text);
        setFilename(file.name);
        onSetFilename(file.name);
      };
      reader.readAsText(file);
    },
    [onSetFilename],
  );

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);

    const file = e.dataTransfer.files?.[0];
    if (file) {
      readFile(file);
    }
  };

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      readFile(file);
    }
  };

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setYaml(e.target.value);
    if (filename) {
      setFilename(null);
      onSetFilename(null);
    }
  };

  const handleAnalyze = () => {
    if (yaml.trim()) {
      onAnalyze(yaml);
    }
  };

  const handleOverlayClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const canAnalyze = yaml.trim().length > 0 && !isLoading;

  return (
    <AnimatePresence>
      <motion.div
        key="yaml-drop-zone-overlay"
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2 }}
        onClick={handleOverlayClick}
        role="dialog"
        aria-modal="true"
        aria-label="Preview Change"
      >
        <motion.div
          className="relative w-full max-w-lg mx-4 rounded-xl border border-border bg-card soft-shadow glass-panel flex flex-col overflow-hidden"
          initial={{ opacity: 0, y: 16, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 12, scale: 0.97 }}
          transition={{ duration: 0.25, ease: 'easeOut' }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-5 py-4 border-b border-border">
            <h2 className="text-base font-semibold text-foreground">Preview Change</h2>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-muted-foreground hover:text-foreground"
              onClick={onClose}
              aria-label="Close"
              disabled={isLoading}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          {/* Body */}
          <div className="flex flex-col gap-4 p-5">
            {/* Drop zone */}
            <div
              className={cn(
                'flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-6 py-8 cursor-pointer transition-colors duration-150 select-none',
                isDragOver
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border bg-muted/30 text-muted-foreground hover:border-primary/60 hover:bg-muted/50',
              )}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              role="button"
              tabIndex={0}
              aria-label="Drop YAML file or click to browse"
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  fileInputRef.current?.click();
                }
              }}
            >
              <Upload
                className={cn(
                  'h-8 w-8 transition-colors duration-150',
                  isDragOver ? 'text-primary' : 'text-muted-foreground',
                )}
              />
              <p className="text-sm font-medium">
                {filename ? (
                  <span className="text-foreground">{filename}</span>
                ) : (
                  <>
                    <span className={cn(isDragOver ? 'text-primary' : 'text-foreground')}>
                      Drop a YAML file
                    </span>{' '}
                    or{' '}
                    <span className="text-primary underline underline-offset-2 cursor-pointer">
                      browse
                    </span>
                  </>
                )}
              </p>
              <p className="text-xs text-muted-foreground">.yaml or .yml files accepted</p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".yaml,.yml,application/x-yaml,text/yaml"
                className="sr-only"
                onChange={handleFileInput}
                tabIndex={-1}
              />
            </div>

            {/* Divider */}
            <div className="flex items-center gap-3">
              <div className="flex-1 h-px bg-border" />
              <span className="text-xs text-muted-foreground uppercase tracking-wide">or paste</span>
              <div className="flex-1 h-px bg-border" />
            </div>

            {/* Textarea */}
            <textarea
              className={cn(
                'w-full min-h-[180px] rounded-md border border-input bg-background px-3 py-2.5',
                'font-mono text-xs text-foreground placeholder:text-muted-foreground',
                'resize-y focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                'transition-colors duration-150',
                isLoading && 'opacity-60 cursor-not-allowed',
              )}
              placeholder={YAML_PLACEHOLDER}
              value={yaml}
              onChange={handleTextareaChange}
              disabled={isLoading}
              spellCheck={false}
              aria-label="YAML manifest content"
            />
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-3 px-5 py-4 border-t border-border bg-muted/20">
            <Button
              variant="outline"
              size="sm"
              onClick={onClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleAnalyze}
              disabled={!canAnalyze}
              className="min-w-[130px]"
            >
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Analyzing…
                </>
              ) : (
                'Analyze Impact'
              )}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

export default YAMLDropZone;
