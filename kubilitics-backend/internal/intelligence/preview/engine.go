package preview

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/kubilitics/kubilitics-backend/internal/graph"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"sigs.k8s.io/yaml"
)

// Engine analyses a YAML manifest against the current cluster graph snapshot
// to predict the blast radius of proposed changes before they are applied.
type Engine struct{}

// NewEngine returns a ready-to-use preview engine.
func NewEngine() *Engine {
	return &Engine{}
}

// parsedResource is an intermediate representation of one Kubernetes resource
// extracted from the manifest YAML.
type parsedResource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec specFragment `json:"spec"`
}

// specFragment captures the fields we need for SPOF / remediation analysis.
type specFragment struct {
	Replicas *int32 `json:"replicas,omitempty"`
}

// AnalyzeManifest parses a (possibly multi-document) YAML manifest, compares
// each resource against the live cluster snapshot, and returns an aggregate
// impact report.
func (e *Engine) AnalyzeManifest(manifest string, snap *graph.GraphSnapshot) (*PreviewResult, error) {
	resources, err := parseMultiDocYAML(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("manifest contains no valid Kubernetes resources")
	}

	result := &PreviewResult{
		AffectedResources: []AffectedResource{},
		NewSPOFs:          []ResourceRef{},
		RemovedSPOFs:      []ResourceRef{},
		Warnings:          []string{},
		Remediations:      []Remediation{},
	}

	var totalBlastScore float64
	healthBefore := computeSimpleHealthScore(snap)
	healthAfter := healthBefore

	for _, res := range resources {
		kind := normalizeKind(res.Kind)
		if kind == "" {
			kind = res.Kind // keep original if unrecognised
		}
		namespace := res.Metadata.Namespace
		if namespace == "" {
			namespace = "default"
		}
		name := res.Metadata.Name

		if name == "" || res.Kind == "" {
			continue // skip resources without required fields
		}

		// Determine if this resource already exists in the snapshot.
		key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)
		_, existsInGraph := snap.Nodes[key]

		impact := "created"
		var blastScore float64

		if existsInGraph {
			impact = "modified"
			// Compute blast radius for the existing resource using the snapshot.
			br, err := snap.ComputeBlastRadius(models.ResourceRef{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
			})
			if err == nil {
				blastScore = br.CriticalityScore
			}
		}

		result.AffectedResources = append(result.AffectedResources, AffectedResource{
			Name:       name,
			Kind:       kind,
			Namespace:  namespace,
			Impact:     impact,
			BlastScore: blastScore,
		})

		totalBlastScore += blastScore

		// --- SPOF detection ---
		e.detectSPOFChanges(res, kind, namespace, name, snap, result)

		// --- Warnings ---
		if existsInGraph && blastScore >= 45 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Modifying %s/%s in %s — this resource has HIGH blast radius (%.0f)",
					kind, name, namespace, blastScore))
		}

		// --- Remediations ---
		e.suggestRemediations(res, kind, namespace, name, snap, result)

		// Approximate health score delta: each high-blast modification degrades health.
		if impact == "modified" && blastScore > 25 {
			healthAfter -= blastScore * 0.1
		}
	}

	// Clamp health scores.
	if healthAfter < 0 {
		healthAfter = 0
	}
	if healthAfter > 100 {
		healthAfter = 100
	}

	result.TotalAffected = len(result.AffectedResources)

	// Aggregate blast radius score: average of per-resource scores (cap at 100).
	if result.TotalAffected > 0 {
		result.BlastRadiusScore = math.Round(totalBlastScore/float64(result.TotalAffected)*100) / 100
		if result.BlastRadiusScore > 100 {
			result.BlastRadiusScore = 100
		}
	}
	result.BlastRadiusLevel = blastRadiusLevel(result.BlastRadiusScore)

	result.HealthScoreBefore = healthBefore
	result.HealthScoreAfter = healthAfter
	result.HealthScoreDelta = healthAfter - healthBefore

	return result, nil
}

// detectSPOFChanges checks whether this manifest resource introduces or removes
// a single-point-of-failure.
func (e *Engine) detectSPOFChanges(
	res parsedResource,
	kind, namespace, name string,
	snap *graph.GraphSnapshot,
	result *PreviewResult,
) {
	key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	// If manifest sets replicas to 1 for a workload, flag as potential new SPOF.
	if res.Spec.Replicas != nil && *res.Spec.Replicas == 1 {
		// Check if it was previously not a SPOF (had replicas > 1).
		if prevReplicas, ok := snap.NodeReplicas[key]; ok && prevReplicas > 1 {
			result.NewSPOFs = append(result.NewSPOFs, ResourceRef{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
			})
		} else if !ok {
			// New resource with 1 replica that has dependents-to-be — flag it.
			result.NewSPOFs = append(result.NewSPOFs, ResourceRef{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
			})
		}
	}

	// If manifest increases replicas from 1 to N > 1, removed SPOF.
	if res.Spec.Replicas != nil && *res.Spec.Replicas > 1 {
		if prevReplicas, ok := snap.NodeReplicas[key]; ok && prevReplicas <= 1 {
			fanIn := len(snap.Reverse[key])
			hasHPA := snap.NodeHasHPA[key]
			if !hasHPA && fanIn > 0 {
				result.RemovedSPOFs = append(result.RemovedSPOFs, ResourceRef{
					Kind:      kind,
					Name:      name,
					Namespace: namespace,
				})
			}
		}
	}
}

// suggestRemediations generates actionable suggestions for the manifest resource.
func (e *Engine) suggestRemediations(
	res parsedResource,
	kind, namespace, name string,
	snap *graph.GraphSnapshot,
	result *PreviewResult,
) {
	key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	// Replica-based remediations.
	if res.Spec.Replicas != nil && *res.Spec.Replicas == 1 {
		result.Remediations = append(result.Remediations, Remediation{
			Type:        "increase-replicas",
			Description: fmt.Sprintf("%s/%s has replicas=1 — increase to at least 3 for redundancy", kind, name),
			Priority:    "high",
		})
	}

	// If modifying an existing resource with no PDB, suggest adding one.
	if _, exists := snap.Nodes[key]; exists {
		if !snap.NodeHasPDB[key] {
			result.Remediations = append(result.Remediations, Remediation{
				Type:        "add-pdb",
				Description: fmt.Sprintf("Add PodDisruptionBudget for %s/%s to protect against voluntary disruptions", kind, name),
				Priority:    "high",
			})
		}
		if !snap.NodeHasHPA[key] {
			result.Remediations = append(result.Remediations, Remediation{
				Type:        "add-hpa",
				Description: fmt.Sprintf("Add HorizontalPodAutoscaler for %s/%s for elastic scaling", kind, name),
				Priority:    "medium",
			})
		}
	}
}

// parseMultiDocYAML splits a multi-document YAML string on "---" separators
// and parses each document into a parsedResource.
func parseMultiDocYAML(manifest string) ([]parsedResource, error) {
	var resources []parsedResource

	reader := bufio.NewReader(strings.NewReader(manifest))
	var current strings.Builder

	flushDoc := func() error {
		doc := strings.TrimSpace(current.String())
		current.Reset()
		if doc == "" {
			return nil
		}
		var res parsedResource
		if err := yaml.Unmarshal([]byte(doc), &res); err != nil {
			return fmt.Errorf("invalid YAML document: %w", err)
		}
		if res.Kind != "" {
			resources = append(resources, res)
		}
		return nil
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if flushErr := flushDoc(); flushErr != nil {
				return nil, flushErr
			}
		} else {
			current.WriteString(line)
		}

		if err == io.EOF {
			if flushErr := flushDoc(); flushErr != nil {
				return nil, flushErr
			}
			break
		}
	}

	return resources, nil
}

// computeSimpleHealthScore derives a rough 0-100 health score from the snapshot.
// In a full implementation this would use the intelligence/health scorer; here we
// compute a lightweight approximation based on SPOF density and PDB coverage.
func computeSimpleHealthScore(snap *graph.GraphSnapshot) float64 {
	if snap.TotalWorkloads == 0 {
		return 100
	}

	spofCount := 0
	pdbCount := 0
	hpaCount := 0

	for key := range snap.Nodes {
		replicas := snap.NodeReplicas[key]
		fanIn := len(snap.Reverse[key])
		hasHPA := snap.NodeHasHPA[key]
		hasPDB := snap.NodeHasPDB[key]

		if replicas <= 1 && !hasHPA && fanIn > 0 {
			spofCount++
		}
		if hasPDB {
			pdbCount++
		}
		if hasHPA {
			hpaCount++
		}
	}

	total := float64(snap.TotalWorkloads)

	// SPOF density (25% weight): 1 - spofCount/total
	spofDensity := 1.0 - float64(spofCount)/total
	if spofDensity < 0 {
		spofDensity = 0
	}

	// PDB coverage (25% weight)
	pdbCoverage := float64(pdbCount) / total
	if pdbCoverage > 1 {
		pdbCoverage = 1
	}

	// HPA coverage (20% weight)
	hpaCoverage := float64(hpaCount) / total
	if hpaCoverage > 1 {
		hpaCoverage = 1
	}

	// Base health (30% weight) — assume decent health
	baseHealth := 0.7

	score := (spofDensity*25 + pdbCoverage*25 + hpaCoverage*20 + baseHealth*30)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// blastRadiusLevel maps a numeric score to a level string.
func blastRadiusLevel(score float64) string {
	switch {
	case score >= 70:
		return "critical"
	case score >= 45:
		return "high"
	case score >= 20:
		return "medium"
	default:
		return "low"
	}
}

// normalizeKind converts common Kubernetes resource kind strings to their
// canonical PascalCase form.
func normalizeKind(kind string) string {
	switch strings.ToLower(kind) {
	case "pod", "pods":
		return "Pod"
	case "deployment", "deployments":
		return "Deployment"
	case "replicaset", "replicasets":
		return "ReplicaSet"
	case "statefulset", "statefulsets":
		return "StatefulSet"
	case "daemonset", "daemonsets":
		return "DaemonSet"
	case "job", "jobs":
		return "Job"
	case "cronjob", "cronjobs":
		return "CronJob"
	case "service", "services":
		return "Service"
	case "configmap", "configmaps":
		return "ConfigMap"
	case "secret", "secrets":
		return "Secret"
	case "ingress", "ingresses":
		return "Ingress"
	case "networkpolicy", "networkpolicies":
		return "NetworkPolicy"
	case "persistentvolumeclaim", "persistentvolumeclaims", "pvc":
		return "PersistentVolumeClaim"
	case "persistentvolume", "persistentvolumes", "pv":
		return "PersistentVolume"
	case "serviceaccount", "serviceaccounts":
		return "ServiceAccount"
	case "horizontalpodautoscaler", "horizontalpodautoscalers", "hpa":
		return "HorizontalPodAutoscaler"
	case "poddisruptionbudget", "poddisruptionbudgets", "pdb":
		return "PodDisruptionBudget"
	case "namespace", "namespaces":
		return "Namespace"
	case "node", "nodes":
		return "Node"
	default:
		return ""
	}
}
