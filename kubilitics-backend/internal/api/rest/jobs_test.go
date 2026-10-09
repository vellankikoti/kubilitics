package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
)

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: jobs.go (retry a failed Job
// by cloning its spec) had zero test coverage despite mutating live
// workload state — a silent bug here either fails to retry or duplicates
// job runs.

func sampleJob(name, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{Name: "worker", Image: "worker:v2"},
					},
				},
			},
		},
		Status: batchv1.JobStatus{
			Failed: 1,
		},
	}
}

func TestHandler_PostJobRetry_Success(t *testing.T) {
	clusterID := "test-cluster"
	job := sampleJob("etl-run", "default")
	client := newCronJobsTestClient(t, toUnstructuredJob(job))
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/default/etl-run/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if kind, _ := created["kind"].(string); kind != "Job" {
		t.Errorf("expected created object kind=Job, got %v", created["kind"])
	}

	// Verify a *new* Job actually exists in the fake cluster (not just the
	// original one echoed back) — there must be exactly 2 Jobs now: the
	// original plus the retry clone.
	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 2 {
		t.Fatalf("expected 2 Jobs (original + retry clone), got %d", len(jobList.Items))
	}

	var clone *unstructured.Unstructured
	for i := range jobList.Items {
		if jobList.Items[i].GetName() != "etl-run" {
			clone = &jobList.Items[i]
		}
	}
	if clone == nil {
		t.Fatal("expected to find the retry clone among the listed jobs")
	}
	if genName := clone.GetGenerateName(); genName != "etl-run-retry-" {
		t.Errorf("expected generateName 'etl-run-retry-', got %q", genName)
	}
	if ns := clone.GetNamespace(); ns != "default" {
		t.Errorf("expected namespace 'default', got %q", ns)
	}
	// Status must NOT be carried over — this is a clone for retry, not a
	// copy of the failed run's terminal state.
	if _, found, _ := unstructured.NestedMap(clone.Object, "status"); found {
		t.Error("expected the retry clone to have no status copied from the failed original")
	}
	// Spec (container image) must be cloned from the original.
	containers, found, err := unstructured.NestedSlice(clone.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("expected 1 container cloned from original spec, found=%v err=%v containers=%v", found, err, containers)
	}
	containerMap, ok := containers[0].(map[string]interface{})
	if !ok || containerMap["image"] != "worker:v2" {
		t.Errorf("expected container image='worker:v2' cloned from original, got %v", containers[0])
	}
}

func TestHandler_PostJobRetry_NotFound(t *testing.T) {
	clusterID := "test-cluster"
	// Seed a different Job so GVR resolution succeeds but the named one 404s.
	client := newCronJobsTestClient(t, toUnstructuredJob(sampleJob("other-job", "default")))
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/default/ghost/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 retrying a nonexistent Job, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostJobRetry_InvalidNamespace(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredJob(sampleJob("etl-run", "default")))
	router := newCronJobsTestHandler(t, client, clusterID)

	// An uppercase namespace is invalid per Kubernetes DNS naming rules
	// (validate.Namespace lowercases then regex-matches, but the request
	// path itself must still resolve to something that 400s, not panic).
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/Invalid_NS!/etl-run/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("expected 400 or 404 for invalid namespace, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostJobRetry_NoSpec(t *testing.T) {
	clusterID := "test-cluster"
	// A Job object missing spec entirely (hand-built to actually produce
	// that shape — the typed struct always has a spec field).
	broken := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      "broken-job",
			"namespace": "default",
		},
	}}
	client := newCronJobsTestClient(t, broken)
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/default/broken-job/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for Job missing spec, got %d: %s", rec.Code, rec.Body.String())
	}

	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Errorf("expected no retry clone created when spec is missing, got %d jobs", len(jobList.Items))
	}
}

func TestHandler_PostJobRetry_RBAC_ViewerForbidden(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredJob(sampleJob("etl-run", "default")))
	router := newCronJobsTestHandlerWithAuth(t, client, clusterID, auth.RoleViewer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/default/etl-run/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer role retrying a Job, got %d: %s", rec.Code, rec.Body.String())
	}

	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Errorf("expected no retry clone created when RBAC denies the viewer, got %d jobs", len(jobList.Items))
	}
}

func TestHandler_PostJobRetry_RBAC_OperatorAllowed(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredJob(sampleJob("etl-run", "default")))
	router := newCronJobsTestHandlerWithAuth(t, client, clusterID, auth.RoleOperator)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/jobs/default/etl-run/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for operator role retrying a Job, got %d: %s", rec.Code, rec.Body.String())
	}
}
