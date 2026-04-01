/* eslint-disable react-refresh/only-export-components */
import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Plus,
  Upload,
  Undo2,
  X,
  Check,
  Copy,
  Download,
  FileCode,
  BookOpen,
  AlertCircle,
  Sparkles,
  Loader2,
  ChevronLeft,
  CheckCircle2,
  Rocket,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { toast } from '@/components/ui/sonner';
import { CodeEditor } from '@/components/editor/CodeEditor';
import { ResourceDocumentation } from '@/components/editor/ResourceDocumentation';
import { cn } from '@/lib/utils';

interface ResourceCreatorProps {
  resourceKind: string;
  defaultYaml: string;
  onClose: () => void;
  onApply: (yaml: string) => void;
  isApplying?: boolean;
  clusterName?: string;
}

interface YamlValidationResult {
  isValid: boolean;
  errors: string[];
}

function validateYaml(yaml: string): YamlValidationResult {
  const errors: string[] = [];

  if (!yaml.trim()) {
    errors.push('YAML cannot be empty');
    return { isValid: false, errors };
  }

  if (!yaml.includes('apiVersion:')) {
    errors.push('Missing required field: apiVersion');
  }
  if (!yaml.includes('kind:')) {
    errors.push('Missing required field: kind');
  }
  if (!yaml.includes('metadata:')) {
    errors.push('Missing required field: metadata');
  }
  if (!yaml.includes('name:')) {
    errors.push('Missing required field: metadata.name');
  }

  if (yaml.includes('\t')) {
    errors.push('Tabs are not allowed in YAML, use spaces');
  }

  return { isValid: errors.length === 0, errors: errors.slice(0, 5) };
}

export function ResourceCreator({
  resourceKind,
  defaultYaml,
  onClose,
  onApply,
  isApplying = false,
  clusterName,
}: ResourceCreatorProps) {
  const navigate = useNavigate();
  const [yaml, setYaml] = useState(defaultYaml);
  const [originalYaml] = useState(defaultYaml);
  const [activeTab, setActiveTab] = useState<'editor' | 'docs'>('editor');
  const [fontSize, setFontSize] = useState<'small' | 'medium' | 'large'>('small');
  const [validation, setValidation] = useState<YamlValidationResult>({ isValid: true, errors: [] });

  const hasChanges = yaml !== originalYaml;

  const handleYamlChange = useCallback((value: string) => {
    setYaml(value);
    setValidation(validateYaml(value));
  }, []);

  const handleUndo = () => {
    setYaml(originalYaml);
    setValidation(validateYaml(originalYaml));
    toast.info('Changes reverted');
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(yaml);
    toast.success('YAML copied to clipboard');
  };

  const handleDownload = () => {
    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${resourceKind.toLowerCase()}.yaml`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 30_000);
    toast.success('YAML downloaded');
  };

  const handleUpload = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.yaml,.yml';
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (file) {
        const reader = new FileReader();
        reader.onload = (event) => {
          const content = event.target?.result as string;
          setYaml(content);
          setValidation(validateYaml(content));
          toast.success(`Loaded ${file.name}`);
        };
        reader.readAsText(file);
      }
    };
    input.click();
  };

  const handleApply = () => {
    if (!validation.isValid) {
      toast.error('Please fix validation errors before applying');
      return;
    }
    onApply(yaml);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 12 }}
      transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
      className="fixed z-[60] flex flex-col bg-white dark:bg-slate-900 overflow-hidden"
      style={{ top: 0, right: 0, bottom: 0, left: 0 }}
      ref={(el) => {
        // Position to cover exactly the main content area (right of sidebar, below header)
        if (el) {
          const main = document.getElementById('main-content');
          if (main) {
            const rect = main.getBoundingClientRect();
            el.style.top = `${rect.top}px`;
            el.style.left = `${rect.left}px`;
            el.style.right = '0px';
            el.style.bottom = '0px';
          }
        }
      }}
    >
        {/* ─── Header — title + tabs + close ──────────────────────────────── */}
        <div className="flex items-center gap-3 px-4 py-2.5 border-b border-slate-200/60 dark:border-slate-700/60 bg-slate-50/80 dark:bg-slate-800/50 shrink-0">
          <Plus className="h-4 w-4 text-primary shrink-0" />
          <span className="text-sm font-semibold text-slate-900 dark:text-slate-100 truncate">Create {resourceKind}</span>
          {clusterName && <span className="text-xs text-muted-foreground font-mono shrink-0">{clusterName}</span>}

          <div className="flex items-center gap-1 ml-3 shrink-0">
            <button
              onClick={() => setActiveTab('editor')}
              className={cn(
                "flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-sm font-medium transition-all border",
                activeTab === 'editor'
                  ? "bg-primary text-primary-foreground border-primary shadow-sm shadow-primary/25"
                  : "bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700 hover:border-primary/50 hover:text-primary"
              )}
            >
              <FileCode className="h-4 w-4" />
              Editor
            </button>
            <button
              onClick={() => setActiveTab('docs')}
              className={cn(
                "flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-sm font-medium transition-all border",
                activeTab === 'docs'
                  ? "bg-primary text-primary-foreground border-primary shadow-sm shadow-primary/25"
                  : "bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700 hover:border-primary/50 hover:text-primary"
              )}
            >
              <BookOpen className="h-4 w-4" />
              Documentation
            </button>
          </div>

          <div className="flex-1" />

          {/* Editor-only toolbar */}
          {activeTab === 'editor' && (
            <div className="flex items-center gap-1 shrink-0">
              <Tooltip>
                <TooltipTrigger asChild>
                  <button onClick={handleCopy} className="h-7 w-7 rounded-lg flex items-center justify-center text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
                    <Copy className="h-3.5 w-3.5" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>Copy</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button onClick={handleDownload} className="h-7 w-7 rounded-lg flex items-center justify-center text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
                    <Download className="h-3.5 w-3.5" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>Download</TooltipContent>
              </Tooltip>
              <button
                onClick={handleUpload}
                className="h-7 rounded-lg border border-slate-200/80 dark:border-slate-700/60 px-2 flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
              >
                <Upload className="h-3 w-3" />
                Import
              </button>
              <div className="w-px h-4 bg-slate-200 dark:bg-slate-700 mx-0.5" />
            </div>
          )}

          <button
            onClick={onClose}
            className="h-7 w-7 rounded-lg flex items-center justify-center text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors shrink-0"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* ─── Editor: validation errors ──────────────────────────────────── */}
        <AnimatePresence>
          {activeTab === 'editor' && !validation.isValid && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.15 }}
              className="overflow-hidden"
            >
              <div className="px-6 py-2.5 bg-red-50/80 dark:bg-red-900/10 border-b border-red-100 dark:border-red-900/20">
                <div className="space-y-1">
                  {validation.errors.map((error, i) => (
                    <p key={i} className="text-xs text-red-600 dark:text-red-400 flex items-center gap-1.5">
                      <span className="h-1 w-1 rounded-full bg-red-400 shrink-0" />
                      {error}
                    </p>
                  ))}
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* ─── Main Content ───────────────────────────────────────────────── */}
        <div className="flex-1 min-h-0 overflow-hidden">
          {activeTab === 'editor' ? (
            <CodeEditor
              value={yaml}
              onChange={handleYamlChange}
              className="h-full rounded-none border-0"
              minHeight="100%"
              fontSize={fontSize}
            />
          ) : (
            <ResourceDocumentation
              resourceKind={resourceKind}
              className="h-full"
            />
          )}
        </div>

        {/* ─── Footer — Editor only ──────────────────────────────────────── */}
        {activeTab === 'editor' && <div className="grid grid-cols-3 items-center px-6 py-4 border-t border-slate-200/60 dark:border-slate-700/60 bg-white dark:bg-slate-900 shadow-[0_-4px_16px_rgba(0,0,0,0.04)] dark:shadow-[0_-4px_16px_rgba(0,0,0,0.2)] shrink-0">
          {/* Left — Cancel + Reset */}
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="flex items-center gap-2 h-10 px-5 rounded-lg text-sm font-semibold border border-red-200 dark:border-red-800/60 text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/15 hover:bg-red-100 dark:hover:bg-red-900/30 hover:border-red-300 dark:hover:border-red-700 transition-colors"
            >
              <X className="h-4 w-4" />
              Cancel
            </button>
            {hasChanges && (
              <button
                onClick={handleUndo}
                className="flex items-center gap-1.5 h-10 px-4 rounded-lg text-sm font-medium text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
              >
                <Undo2 className="h-4 w-4" />
                Reset
              </button>
            )}
          </div>

          {/* Center — Validation status */}
          <div className="flex justify-center">
            {validation.isValid && yaml.trim() ? (
              <span className="flex items-center gap-2 h-10 px-5 rounded-lg text-sm font-semibold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800">
                <CheckCircle2 className="h-5 w-5" />
                Valid YAML
              </span>
            ) : !yaml.trim() ? null : (
              <span className="flex items-center gap-2 h-10 px-5 rounded-lg text-sm font-semibold text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                <AlertCircle className="h-5 w-5" />
                {validation.errors.length} error{validation.errors.length !== 1 ? 's' : ''}
              </span>
            )}
          </div>

          {/* Right — Create */}
          <div className="flex justify-end">
            <button
              onClick={handleApply}
              disabled={!validation.isValid || isApplying}
              className={cn(
                "flex items-center gap-2 h-10 px-6 rounded-lg text-sm font-semibold transition-all",
                validation.isValid && !isApplying
                  ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-md shadow-primary/30"
                  : "bg-slate-200 dark:bg-slate-700 text-slate-400 dark:text-slate-500 cursor-not-allowed"
              )}
            >
              {isApplying ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Applying...
                </>
              ) : (
                <>
                  <Rocket className="h-4 w-4" />
                  Create {resourceKind}
                </>
              )}
            </button>
          </div>
        </div>}
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Default YAML templates for different resource types
// Each template is production-ready with real names, images, and best practices
// ---------------------------------------------------------------------------
export const DEFAULT_YAMLS: Record<string, string> = {
  Pod: `apiVersion: v1
kind: Pod
metadata:
  name: my-web-app
  namespace: default
  labels:
    app: my-web-app
    env: production
spec:
  containers:
    - name: web
      image: nginx:alpine
      ports:
        - containerPort: 80
          name: http
      # Resource requests and limits ensure fair scheduling and prevent OOM kills
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "250m"
      # Readiness probe: traffic is routed only after this succeeds
      readinessProbe:
        httpGet:
          path: /
          port: 80
        initialDelaySeconds: 5
        periodSeconds: 10
      # Liveness probe: container is restarted if this fails
      livenessProbe:
        httpGet:
          path: /
          port: 80
        initialDelaySeconds: 15
        periodSeconds: 20
      imagePullPolicy: IfNotPresent
  restartPolicy: Always`,

  Deployment: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-web-app
  namespace: default
  labels:
    app: my-web-app
    env: production
spec:
  replicas: 3
  # Rolling update strategy for zero-downtime deployments
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: my-web-app
  template:
    metadata:
      labels:
        app: my-web-app
        env: production
    spec:
      containers:
        - name: web
          image: nginx:alpine
          ports:
            - containerPort: 80
              name: http
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 20`,

  Service: `apiVersion: v1
kind: Service
metadata:
  name: my-web-app
  namespace: default
  labels:
    app: my-web-app
spec:
  # Types: ClusterIP (internal), NodePort (node access), LoadBalancer (cloud LB)
  type: ClusterIP
  selector:
    app: my-web-app
  ports:
    - name: http
      protocol: TCP
      port: 80
      targetPort: 80`,

  ConfigMap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  # Simple key-value pairs
  APP_ENV: production
  LOG_LEVEL: info
  # Multi-line config file
  nginx.conf: |
    server {
      listen 80;
      server_name localhost;
      location / {
        root /usr/share/nginx/html;
      }
    }`,

  Secret: `apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: default
  labels:
    app: my-web-app
# Type: Opaque (generic), kubernetes.io/tls (TLS cert), kubernetes.io/dockerconfigjson (registry)
type: Opaque
# Use stringData for plain text (auto-encoded to base64 by Kubernetes)
stringData:
  DB_PASSWORD: changeme-use-a-real-password
  API_KEY: your-api-key-here
# Or use data with base64-encoded values:
# data:
#   DB_PASSWORD: Y2hhbmdlbWU=    # echo -n "changeme" | base64`,

  StatefulSet: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: default
  labels:
    app: postgres
spec:
  # serviceName must match a headless Service for stable DNS names
  # Each pod gets: postgres-0.postgres-headless.default.svc.cluster.local
  serviceName: postgres-headless
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
              name: postgres
          env:
            - name: POSTGRES_DB
              value: mydb
            - name: POSTGRES_USER
              value: admin
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "1"
          readinessProbe:
            exec:
              command: ["pg_isready", "-U", "admin"]
            initialDelaySeconds: 10
            periodSeconds: 5
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  # Each replica gets its own PVC for stable persistent storage
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi`,

  DaemonSet: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-collector
  namespace: kube-system
  labels:
    app: log-collector
spec:
  selector:
    matchLabels:
      app: log-collector
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        app: log-collector
    spec:
      # Tolerations allow scheduling on control-plane and tainted nodes
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
      containers:
        - name: log-collector
          image: busybox:1.36
          command: ["sh", "-c", "tail -f /var/log/syslog || tail -f /dev/null"]
          resources:
            requests:
              memory: "32Mi"
              cpu: "50m"
            limits:
              memory: "64Mi"
              cpu: "100m"
          volumeMounts:
            - name: varlog
              mountPath: /var/log
              readOnly: true
      volumes:
        - name: varlog
          hostPath:
            path: /var/log`,

  Job: `apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  namespace: default
  labels:
    app: db-migration
spec:
  # Number of successful completions required
  completions: 1
  # How many pods to run in parallel
  parallelism: 1
  # Number of retries before marking the Job as failed
  backoffLimit: 3
  # Automatically clean up after 1 hour (seconds)
  ttlSecondsAfterFinished: 3600
  template:
    metadata:
      labels:
        app: db-migration
    spec:
      containers:
        - name: migrate
          image: busybox:1.36
          command: ["sh", "-c", "echo 'Running database migration...' && sleep 5 && echo 'Migration complete!'"]
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "250m"
      restartPolicy: Never`,

  CronJob: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-backup
  namespace: default
  labels:
    app: nightly-backup
spec:
  # Schedule format: minute hour day-of-month month day-of-week
  # Examples: "0 2 * * *" (daily 2am), "*/15 * * * *" (every 15 min), "0 0 * * 0" (weekly Sun)
  schedule: "0 2 * * *"
  # Forbid: skip if previous still running; Replace: stop previous and start new
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  # Optional: deadline to start the job (seconds). Missed if not started in time.
  startingDeadlineSeconds: 600
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        metadata:
          labels:
            app: nightly-backup
        spec:
          containers:
            - name: backup
              image: busybox:1.36
              command: ["sh", "-c", "echo 'Starting backup at $(date)' && sleep 10 && echo 'Backup completed'"]
              resources:
                requests:
                  memory: "64Mi"
                  cpu: "100m"
                limits:
                  memory: "256Mi"
                  cpu: "500m"
          restartPolicy: OnFailure`,

  Ingress: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-web-app-ingress
  namespace: default
  labels:
    app: my-web-app
  annotations:
    # Uncomment for nginx ingress controller options:
    # nginx.ingress.kubernetes.io/rewrite-target: /
    # nginx.ingress.kubernetes.io/ssl-redirect: "true"
    # nginx.ingress.kubernetes.io/proxy-body-size: "10m"
spec:
  # Uncomment ingressClassName if you have multiple ingress controllers
  # ingressClassName: nginx
  # TLS configuration — uncomment to enable HTTPS
  # tls:
  #   - hosts:
  #       - app.example.com
  #     secretName: app-tls-secret
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-web-app
                port:
                  number: 80
          # Additional path example:
          # - path: /api
          #   pathType: Prefix
          #   backend:
          #     service:
          #       name: backend-api
          #       port:
          #         number: 8080`,

  PersistentVolumeClaim: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data
  namespace: default
  labels:
    app: my-web-app
spec:
  # Access modes: ReadWriteOnce (single node), ReadOnlyMany (multi read), ReadWriteMany (multi r/w)
  accessModes:
    - ReadWriteOnce
  # Omit storageClassName for cluster default, or specify explicitly
  storageClassName: standard
  resources:
    requests:
      storage: 5Gi`,

  Namespace: `apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  labels:
    name: my-namespace
    env: production`,

  ServiceAccount: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: app-service-account
  namespace: default
  labels:
    app: my-web-app
  annotations:
    # AWS IAM Role for Service Accounts (IRSA):
    # eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/my-role
    #
    # GCP Workload Identity:
    # iam.gke.io/gcp-service-account: my-sa@my-project.iam.gserviceaccount.com
automountServiceAccountToken: true`,

  Role: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: default
  labels:
    app: my-web-app
# Each rule grants access to specific API resources
rules:
  # Read-only access to pods
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["get", "list", "watch"]
  # Read-only access to configmaps and secrets
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list"]
  # Full access to deployments in the apps API group
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]`,

  ClusterRole: `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-monitoring
  labels:
    app: monitoring
# ClusterRoles are not namespaced - they apply cluster-wide
rules:
  # Read-only access to all pods and services across all namespaces
  - apiGroups: [""]
    resources: ["pods", "services", "endpoints", "namespaces"]
    verbs: ["get", "list", "watch"]
  # Read-only access to deployments, statefulsets, daemonsets
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
    verbs: ["get", "list", "watch"]
  # Read node metrics
  - apiGroups: ["metrics.k8s.io"]
    resources: ["pods", "nodes"]
    verbs: ["get", "list"]`,

  RoleBinding: `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pod-reader-binding
  namespace: default
  labels:
    app: my-web-app
# Subjects: who gets access (ServiceAccount, User, or Group)
subjects:
  - kind: ServiceAccount
    name: app-service-account
    namespace: default
  # Uncomment to bind to a user or group:
  # - kind: User
  #   name: jane@example.com
  #   apiGroup: rbac.authorization.k8s.io
# roleRef: which Role or ClusterRole to bind
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io`,

  NetworkPolicy: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: backend-network-policy
  namespace: default
  labels:
    app: backend-service
spec:
  # Selects which pods this policy applies to
  podSelector:
    matchLabels:
      app: backend-service
  policyTypes:
    - Ingress
    - Egress
  # Allow incoming traffic only from frontend pods on port 8080
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: frontend
        # Uncomment to also restrict by namespace:
        # - namespaceSelector:
        #     matchLabels:
        #       env: production
      ports:
        - protocol: TCP
          port: 8080
  # Allow outgoing traffic to database and DNS
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432
    # Allow DNS resolution (required for most workloads)
    - to: []
      ports:
        - protocol: UDP
          port: 53
        - protocol: TCP
          port: 53`,

  HorizontalPodAutoscaler: `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-web-app-hpa
  namespace: default
  labels:
    app: my-web-app
spec:
  # Target the Deployment to scale
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-web-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
    # Scale based on CPU utilization
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    # Scale based on memory utilization
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
    scaleUp:
      stabilizationWindowSeconds: 60`,

  ClusterRoleBinding: `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cluster-monitoring-binding
  labels:
    app: monitoring
subjects:
  - kind: ServiceAccount
    name: monitoring-service-account
    namespace: monitoring
  # Uncomment for group binding:
  # - kind: Group
  #   name: platform-team
  #   apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-monitoring
  apiGroup: rbac.authorization.k8s.io`,

  PersistentVolume: `apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /data`,

  StorageClass: `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer`,

  VolumeSnapshot: `apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: data-snapshot
  namespace: default
spec:
  source:
    persistentVolumeClaimName: app-data
  volumeSnapshotClassName: default-snapclass`,

  VolumeSnapshotClass: `apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: default-snapclass
driver: disk.csi.cloud.com
deletionPolicy: Delete`,

  ResourceQuota: `apiVersion: v1
kind: ResourceQuota
metadata:
  name: team-quota
  namespace: default
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 4Gi
    limits.cpu: "8"
    limits.memory: 8Gi
    pods: "20"`,

  LimitRange: `apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: default
spec:
  limits:
    - default:
        cpu: "500m"
        memory: 512Mi
      defaultRequest:
        cpu: "100m"
        memory: 128Mi
      type: Container`,

  PodDisruptionBudget: `apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-web-app-pdb
  namespace: default
  labels:
    app: my-web-app
spec:
  # Ensure at least 2 pods remain available during voluntary disruptions
  # (node drains, cluster upgrades, etc.)
  # Use minAvailable OR maxUnavailable, not both
  minAvailable: 2
  # Alternative: maxUnavailable: 1
  selector:
    matchLabels:
      app: my-web-app`,

  PriorityClass: `apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000
globalDefault: false
description: High priority for production workloads`,

  ReplicaSet: `apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: my-web-app-rs
  namespace: default
  labels:
    app: my-web-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-web-app
  template:
    metadata:
      labels:
        app: my-web-app
    spec:
      containers:
        - name: web
          image: nginx:alpine`,

  Endpoints: `apiVersion: v1
kind: Endpoints
metadata:
  name: external-db
  namespace: default
subsets:
  - addresses:
      - ip: 10.0.0.1
    ports:
      - port: 5432`,

  EndpointSlice: `apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: my-web-app-slice
  namespace: default
  labels:
    kubernetes.io/service-name: my-web-app
addressType: IPv4
ports:
  - port: 80
endpoints:
  - addresses:
      - 10.0.0.1`,

  IngressClass: `apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx
spec:
  controller: nginx.org/ingress-controller`,

  VolumeAttachment: `apiVersion: storage.k8s.io/v1
kind: VolumeAttachment
metadata:
  name: csi-vol-attach
spec:
  attacher: kubernetes.io/csi
  nodeName: worker-node-1
  source:
    persistentVolumeName: pv-data`,

  Lease: `apiVersion: coordination.k8s.io/v1
kind: Lease
metadata:
  name: leader-election
  namespace: default
spec:
  holderIdentity: controller-pod-xyz
  leaseDurationSeconds: 40`,

  VerticalPodAutoscaler: `apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-web-app-vpa
  namespace: default
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-web-app
  updatePolicy:
    updateMode: "Auto"`,

  ReplicationController: `apiVersion: v1
kind: ReplicationController
metadata:
  name: my-web-app-rc
  namespace: default
spec:
  replicas: 3
  selector:
    app: my-web-app
  template:
    metadata:
      labels:
        app: my-web-app
    spec:
      containers:
        - name: web
          image: nginx:alpine`,

  CustomResourceDefinition: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
  scope: Namespaced
  names:
    plural: widgets
    singular: widget
    kind: Widget`,

  RuntimeClass: `apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: gvisor
handler: runsc`,

  ValidatingWebhookConfiguration: `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: pod-validation
webhooks:
  - name: validate.pods.example.com
    clientConfig:
      service:
        name: webhook-service
        namespace: webhook-system
        port: 443
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE"]
        resources: ["pods"]
    admissionReviewVersions: ["v1"]
    sideEffects: None
    failurePolicy: Fail`,

  MutatingWebhookConfiguration: `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: pod-mutation
webhooks:
  - name: mutate.pods.example.com
    clientConfig:
      service:
        name: webhook-service
        namespace: webhook-system
        port: 443
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE"]
        resources: ["pods"]
    admissionReviewVersions: ["v1"]
    sideEffects: None
    failurePolicy: Fail`,

  APIService: `apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v1beta1.metrics.k8s.io
spec:
  service:
    namespace: kube-system
    name: metrics-server
  group: metrics.k8s.io
  version: v1beta1
  insecureSkipTLSVerify: false`,

  PodTemplate: `apiVersion: v1
kind: PodTemplate
metadata:
  name: web-template
  namespace: default
template:
  metadata:
    labels:
      app: my-web-app
  spec:
    containers:
      - name: web
        image: nginx:alpine`,

  ControllerRevision: `apiVersion: apps/v1
kind: ControllerRevision
metadata:
  name: my-app-revision-1
  namespace: default
revision: 1
data:
  # Managed by StatefulSet or DaemonSet
  # Provide the underlying object state here
  {}`,

  ResourceSlice: `apiVersion: resource.k8s.io/v1alpha3
kind: ResourceSlice
metadata:
  name: gpu-slice-0
spec:
  driverName: gpu.example.com
  pool:
    name: gpu-pool
    generation: 0
    resourceSliceCount: 1`,

  DeviceClass: `apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: gpu-class
spec:
  selectors:
    - cel:
        expression: "device.driver == 'example.com/driver'"`
};

// ---------------------------------------------------------------------------
// Example YAML templates — multiple real-world patterns per resource type
// ---------------------------------------------------------------------------
export const EXAMPLE_YAMLS: Record<string, { title: string; yaml: string }[]> = {
  Pod: [
    {
      title: 'Single Container',
      yaml: `apiVersion: v1
kind: Pod
metadata:
  name: my-web-app
  namespace: default
  labels:
    app: my-web-app
spec:
  containers:
    - name: web
      image: nginx:alpine
      ports:
        - containerPort: 80
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "250m"
      readinessProbe:
        httpGet:
          path: /
          port: 80
        initialDelaySeconds: 5
        periodSeconds: 10
      livenessProbe:
        httpGet:
          path: /
          port: 80
        initialDelaySeconds: 15
        periodSeconds: 20`,
    },
    {
      title: 'Multi Container (Sidecar)',
      yaml: `apiVersion: v1
kind: Pod
metadata:
  name: app-with-sidecar
  namespace: default
  labels:
    app: app-with-sidecar
spec:
  containers:
    # Main application container
    - name: app
      image: nginx:alpine
      ports:
        - containerPort: 80
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "250m"
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/nginx
    # Sidecar: ships logs to a central logging system
    - name: log-shipper
      image: busybox:1.36
      command: ["sh", "-c", "tail -F /var/log/nginx/access.log"]
      resources:
        requests:
          memory: "32Mi"
          cpu: "50m"
        limits:
          memory: "64Mi"
          cpu: "100m"
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/nginx
          readOnly: true
  volumes:
    - name: shared-logs
      emptyDir: {}`,
    },
    {
      title: 'Init Container',
      yaml: `apiVersion: v1
kind: Pod
metadata:
  name: app-with-init
  namespace: default
  labels:
    app: app-with-init
spec:
  # Init containers run before main containers and must succeed first
  initContainers:
    - name: wait-for-db
      image: busybox:1.36
      command: ["sh", "-c", "until nc -z postgres-headless 5432; do echo waiting for db; sleep 2; done"]
      resources:
        requests:
          memory: "32Mi"
          cpu: "50m"
        limits:
          memory: "64Mi"
          cpu: "100m"
  containers:
    - name: app
      image: nginx:alpine
      ports:
        - containerPort: 80
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "250m"`,
    },
  ],

  Deployment: [
    {
      title: 'Basic Web App',
      yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-web-app
  namespace: default
  labels:
    app: my-web-app
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: my-web-app
  template:
    metadata:
      labels:
        app: my-web-app
    spec:
      containers:
        - name: web
          image: nginx:alpine
          ports:
            - containerPort: 80
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 20`,
    },
    {
      title: 'With ConfigMap & Secret Mounts',
      yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-service
  namespace: default
  labels:
    app: backend-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend-service
  template:
    metadata:
      labels:
        app: backend-service
    spec:
      containers:
        - name: app
          image: nginx:alpine
          ports:
            - containerPort: 8080
          # Environment variables from ConfigMap and Secret
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          # Mount config files from ConfigMap
          volumeMounts:
            - name: config-volume
              mountPath: /etc/app/config
              readOnly: true
            - name: secret-volume
              mountPath: /etc/app/secrets
              readOnly: true
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
      volumes:
        - name: config-volume
          configMap:
            name: app-config
        - name: secret-volume
          secret:
            secretName: app-secrets`,
    },
    {
      title: 'Blue-Green Deployment',
      yaml: `# Blue-Green pattern: deploy "green" alongside "blue", then switch Service selector
# Step 1: Deploy green version with a distinct label
# Step 2: Test green independently
# Step 3: Update Service selector from version: blue -> version: green
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-web-app-green
  namespace: default
  labels:
    app: my-web-app
    version: green
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-web-app
      version: green
  template:
    metadata:
      labels:
        app: my-web-app
        version: green
    spec:
      containers:
        - name: web
          image: nginx:alpine
          ports:
            - containerPort: 80
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 5`,
    },
  ],

  Service: [
    {
      title: 'ClusterIP (Internal)',
      yaml: `apiVersion: v1
kind: Service
metadata:
  name: backend-service
  namespace: default
spec:
  type: ClusterIP
  selector:
    app: backend-service
  ports:
    - name: http
      port: 80
      targetPort: 8080`,
    },
    {
      title: 'NodePort (External via Node)',
      yaml: `apiVersion: v1
kind: Service
metadata:
  name: my-web-app-nodeport
  namespace: default
spec:
  type: NodePort
  selector:
    app: my-web-app
  ports:
    - name: http
      port: 80
      targetPort: 80
      # nodePort range: 30000-32767 (omit for auto-assign)
      nodePort: 30080`,
    },
    {
      title: 'LoadBalancer (Cloud)',
      yaml: `apiVersion: v1
kind: Service
metadata:
  name: my-web-app-lb
  namespace: default
  annotations:
    # AWS NLB example:
    # service.beta.kubernetes.io/aws-load-balancer-type: nlb
    # GCP internal LB example:
    # networking.gke.io/load-balancer-type: Internal
spec:
  type: LoadBalancer
  selector:
    app: my-web-app
  ports:
    - name: http
      port: 80
      targetPort: 80
    - name: https
      port: 443
      targetPort: 443`,
    },
    {
      title: 'Headless Service (StatefulSet DNS)',
      yaml: `# Headless Service: no ClusterIP, DNS returns pod IPs directly
# Required for StatefulSets to give each pod a stable DNS name
apiVersion: v1
kind: Service
metadata:
  name: postgres-headless
  namespace: default
spec:
  type: ClusterIP
  clusterIP: None
  selector:
    app: postgres
  ports:
    - name: postgres
      port: 5432
      targetPort: 5432`,
    },
  ],

  StatefulSet: [
    {
      title: 'PostgreSQL Cluster',
      yaml: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: default
spec:
  serviceName: postgres-headless
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_DB
              value: mydb
            - name: POSTGRES_USER
              value: admin
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "1"
          readinessProbe:
            exec:
              command: ["pg_isready", "-U", "admin"]
            initialDelaySeconds: 10
            periodSeconds: 5
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi`,
    },
    {
      title: 'Redis Cluster',
      yaml: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: default
spec:
  serviceName: redis-headless
  replicas: 3
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          ports:
            - containerPort: 6379
              name: redis
          command: ["redis-server", "--appendonly", "yes"]
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
          readinessProbe:
            exec:
              command: ["redis-cli", "ping"]
            initialDelaySeconds: 5
            periodSeconds: 5
          volumeMounts:
            - name: data
              mountPath: /data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 5Gi`,
    },
  ],

  Job: [
    {
      title: 'Database Migration',
      yaml: `apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  namespace: default
spec:
  completions: 1
  backoffLimit: 3
  ttlSecondsAfterFinished: 3600
  template:
    spec:
      containers:
        - name: migrate
          image: busybox:1.36
          command: ["sh", "-c", "echo 'Running migration...' && sleep 5 && echo 'Done'"]
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "250m"
      restartPolicy: Never`,
    },
    {
      title: 'Parallel Batch Processing',
      yaml: `apiVersion: batch/v1
kind: Job
metadata:
  name: batch-processor
  namespace: default
spec:
  # Process 10 items total, 3 at a time
  completions: 10
  parallelism: 3
  backoffLimit: 5
  template:
    spec:
      containers:
        - name: worker
          image: busybox:1.36
          command: ["sh", "-c", "echo Processing item... && sleep 10 && echo Done"]
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
      restartPolicy: Never`,
    },
  ],

  CronJob: [
    {
      title: 'Nightly Backup',
      yaml: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-backup
  namespace: default
spec:
  schedule: "0 2 * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        spec:
          containers:
            - name: backup
              image: busybox:1.36
              command: ["sh", "-c", "echo 'Backup started' && sleep 10 && echo 'Backup complete'"]
              resources:
                requests:
                  memory: "64Mi"
                  cpu: "100m"
                limits:
                  memory: "256Mi"
                  cpu: "500m"
          restartPolicy: OnFailure`,
    },
    {
      title: 'Cache Warmer (Every 15 Minutes)',
      yaml: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: cache-warmer
  namespace: default
spec:
  schedule: "*/15 * * * *"
  concurrencyPolicy: Replace
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 3
  startingDeadlineSeconds: 300
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: warmer
              image: busybox:1.36
              command: ["sh", "-c", "echo 'Warming cache...' && sleep 5 && echo 'Done'"]
              resources:
                requests:
                  memory: "32Mi"
                  cpu: "50m"
                limits:
                  memory: "64Mi"
                  cpu: "100m"
          restartPolicy: OnFailure`,
    },
  ],

  Ingress: [
    {
      title: 'Simple Host-Based Routing',
      yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
  namespace: default
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-web-app
                port:
                  number: 80`,
    },
    {
      title: 'TLS with Path-Based Routing',
      yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress-tls
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - app.example.com
      secretName: app-tls-secret
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: backend-api
                port:
                  number: 8080`,
    },
    {
      title: 'Multi-Host (Fanout)',
      yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-host-ingress
  namespace: default
spec:
  ingressClassName: nginx
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: backend-api
                port:
                  number: 8080
    - host: admin.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: admin-panel
                port:
                  number: 3000`,
    },
  ],
};
