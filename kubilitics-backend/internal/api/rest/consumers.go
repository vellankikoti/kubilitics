package rest

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
)

// resolveDeploymentRefs resolves each pod's owning Deployment (via its
// ReplicaSet owner reference) using a single namespace-scoped ReplicaSet
// List call, instead of one ReplicaSets().Get() per matching pod (the
// previous N+1 pattern duplicated across all three consumer handlers below).
func resolveDeploymentRefs(ctx context.Context, client *k8s.Client, namespace string, pods []corev1.Pod) []Ref {
	rsNames := make(map[string]struct{})
	for _, pod := range pods {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "ReplicaSet" {
				rsNames[ref.Name] = struct{}{}
				break
			}
		}
	}
	if len(rsNames) == 0 {
		return nil
	}
	rsList, err := client.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	rsToDeployment := make(map[string]string, len(rsList.Items))
	for _, rs := range rsList.Items {
		for _, o := range rs.OwnerReferences {
			if o.Kind == "Deployment" {
				rsToDeployment[rs.Name] = o.Name
				break
			}
		}
	}
	var out []Ref
	seenDep := make(map[string]bool)
	for _, pod := range pods {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind != "ReplicaSet" {
				continue
			}
			if depName, ok := rsToDeployment[ref.Name]; ok {
				k := pod.Namespace + "/" + depName
				if !seenDep[k] {
					seenDep[k] = true
					out = append(out, Ref{Namespace: pod.Namespace, Name: depName})
				}
			}
			break
		}
	}
	return out
}

// ConsumersResponse is the JSON shape for configmap/secret consumers endpoints.
type ConsumersResponse struct {
	Pods         []Ref `json:"pods"`
	Deployments  []Ref `json:"deployments"`
	StatefulSets []Ref `json:"statefulSets"`
	DaemonSets   []Ref `json:"daemonSets"`
	Jobs         []Ref `json:"jobs"`
	CronJobs     []Ref `json:"cronJobs"`
}

// Ref is a namespace/name reference.
type Ref struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func podReferencesConfigMap(pod *corev1.Pod, configMapName string) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.ConfigMap != nil && vol.ConfigMap.Name == configMapName {
			return true
		}
	}
	for _, c := range pod.Spec.Containers {
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name == configMapName {
				return true
			}
		}
		for _, env := range c.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == configMapName {
				return true
			}
		}
	}
	return false
}

func podReferencesSecret(pod *corev1.Pod, secretName string) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			return true
		}
	}
	for _, c := range pod.Spec.Containers {
		for _, ef := range c.EnvFrom {
			if ef.SecretRef != nil && ef.SecretRef.Name == secretName {
				return true
			}
		}
		for _, env := range c.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secretName {
				return true
			}
		}
	}
	return false
}

// podReferencesPVC returns true if the pod uses the given PVC (by claim name) in the same namespace.
func podReferencesPVC(pod *corev1.Pod, claimName string) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == claimName {
			return true
		}
	}
	return false
}

func buildConsumersFromPods(pods []corev1.Pod) ConsumersResponse {
	seenPods := make(map[string]bool)
	seenStatefulSets := make(map[string]bool)
	seenDaemonSets := make(map[string]bool)
	seenJobs := make(map[string]bool)
	seenCronJobs := make(map[string]bool)

	var out ConsumersResponse
	for i := range pods {
		pod := &pods[i]
		key := pod.Namespace + "/" + pod.Name
		if !seenPods[key] {
			seenPods[key] = true
			out.Pods = append(out.Pods, Ref{Namespace: pod.Namespace, Name: pod.Name})
		}
		for _, ref := range pod.OwnerReferences {
			switch ref.Kind {
			case "ReplicaSet":
				// Deployments own ReplicaSets; we aggregate by Deployment
				// For simplicity we only add ReplicaSet's owner Deployment if we have it; else skip
				continue
			case "StatefulSet":
				k := pod.Namespace + "/" + ref.Name
				if !seenStatefulSets[k] {
					seenStatefulSets[k] = true
					out.StatefulSets = append(out.StatefulSets, Ref{Namespace: pod.Namespace, Name: ref.Name})
				}
			case "DaemonSet":
				k := pod.Namespace + "/" + ref.Name
				if !seenDaemonSets[k] {
					seenDaemonSets[k] = true
					out.DaemonSets = append(out.DaemonSets, Ref{Namespace: pod.Namespace, Name: ref.Name})
				}
			case "Job":
				k := pod.Namespace + "/" + ref.Name
				if !seenJobs[k] {
					seenJobs[k] = true
					out.Jobs = append(out.Jobs, Ref{Namespace: pod.Namespace, Name: ref.Name})
				}
			case "CronJob":
				k := pod.Namespace + "/" + ref.Name
				if !seenCronJobs[k] {
					seenCronJobs[k] = true
					out.CronJobs = append(out.CronJobs, Ref{Namespace: pod.Namespace, Name: ref.Name})
				}
			}
		}
	}
	// Resolve ReplicaSets to Deployments by fetching RS owner (optional; we can add later)
	// For now we leave Deployments empty when only ReplicaSet is owner; frontend can still show pod count.
	// Alternatively: for each pod with owner ReplicaSet, get RS and add RS.OwnerReferences Deployment.
	// Doing that would require client in buildConsumersFromPods - so we do a second pass in the handler.
	return out
}

// GetConfigMapConsumers handles GET /clusters/{clusterId}/resources/configmaps/{namespace}/{name}/consumers
func (h *Handler) GetConfigMapConsumers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	namespace := vars["namespace"]
	name := vars["name"]
	if !validate.ClusterID(clusterID) || !validate.Namespace(namespace) || !validate.Name(name) {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid clusterId, namespace, or name", requestID)
		return
	}
	// Headlamp/Lens model: try kubeconfig from request first, fall back to stored cluster
	client, err := h.getClientFromRequest(r.Context(), r, clusterID, h.cfg)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	podList, err := client.Clientset.CoreV1().Pods(namespace).List(r.Context(), metav1.ListOptions{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var matching []corev1.Pod
	for i := range podList.Items {
		if podReferencesConfigMap(&podList.Items[i], name) {
			matching = append(matching, podList.Items[i])
		}
	}
	out := buildConsumersFromPods(matching)
	out.Deployments = resolveDeploymentRefs(r.Context(), client, namespace, matching)
	respondJSON(w, http.StatusOK, out)
}

// GetSecretConsumers handles GET /clusters/{clusterId}/resources/secrets/{namespace}/{name}/consumers
func (h *Handler) GetSecretConsumers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	namespace := vars["namespace"]
	name := vars["name"]
	if !validate.ClusterID(clusterID) || !validate.Namespace(namespace) || !validate.Name(name) {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid clusterId, namespace, or name", requestID)
		return
	}
	// Headlamp/Lens model: try kubeconfig from request first, fall back to stored cluster
	client, err := h.getClientFromRequest(r.Context(), r, clusterID, h.cfg)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	podList, err := client.Clientset.CoreV1().Pods(namespace).List(r.Context(), metav1.ListOptions{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var matching []corev1.Pod
	for i := range podList.Items {
		if podReferencesSecret(&podList.Items[i], name) {
			matching = append(matching, podList.Items[i])
		}
	}
	out := buildConsumersFromPods(matching)
	out.Deployments = resolveDeploymentRefs(r.Context(), client, namespace, matching)
	respondJSON(w, http.StatusOK, out)
}

// GetPVCConsumers handles GET /clusters/{clusterId}/resources/persistentvolumeclaims/{namespace}/{name}/consumers
// Returns pods (and their owning workloads) that reference this PVC via spec.volumes[].persistentVolumeClaim.claimName.
func (h *Handler) GetPVCConsumers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	namespace := vars["namespace"]
	name := vars["name"]
	if !validate.ClusterID(clusterID) || !validate.Namespace(namespace) || !validate.Name(name) {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid clusterId, namespace, or name", requestID)
		return
	}
	// Headlamp/Lens model: try kubeconfig from request first, fall back to stored cluster
	client, err := h.getClientFromRequest(r.Context(), r, clusterID, h.cfg)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	podList, err := client.Clientset.CoreV1().Pods(namespace).List(r.Context(), metav1.ListOptions{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var matching []corev1.Pod
	for i := range podList.Items {
		if podReferencesPVC(&podList.Items[i], name) {
			matching = append(matching, podList.Items[i])
		}
	}
	out := buildConsumersFromPods(matching)
	out.Deployments = resolveDeploymentRefs(r.Context(), client, namespace, matching)
	respondJSON(w, http.StatusOK, out)
}
