package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: cronjobs.go (trigger a
// one-off Job from a CronJob's spec.jobTemplate, list a CronJob's child
// Jobs) had zero test coverage despite mutating live workload state — a
// silent bug here duplicates job runs or triggers the wrong template.

// newCronJobsTestClient builds a *k8s.Client with a working dynamic client
// seeded with the given objects. Both CronJobTrigger and GetCronJobJobs
// drive everything through client.Dynamic (GetResource/ListResources/
// CreateResource), never through the typed Clientset.
func newCronJobsTestClient(t *testing.T, objects ...runtime.Object) *k8s.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add batchv1 to scheme: %v", err)
	}
	dyn := dynamicfake.NewSimpleDynamicClient(scheme, objects...)
	client := k8s.NewClientForTest(fake.NewSimpleClientset())
	client.Dynamic = dyn
	return client
}

func newCronJobsTestHandler(t *testing.T, client *k8s.Client, clusterID string) *mux.Router {
	t.Helper()
	cluster := &models.Cluster{ID: clusterID, Name: clusterID, Context: "test-ctx", Status: "connected"}
	mockService := &mockClusterServiceWithClient{clusters: []*models.Cluster{cluster}, client: client}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

// newCronJobsTestHandlerWithAuth wires RBAC through middleware.RequireOperator
// / RequireViewer by enabling auth and injecting claims directly into the
// request context (the same pattern TestHandler_ListClusters_WithPermissions
// in cluster_handler_test.go uses), rather than minting a real JWT.
func newCronJobsTestHandlerWithAuth(t *testing.T, client *k8s.Client, clusterID, role string) *mux.Router {
	t.Helper()
	repo := setupTestRepoForAuth(t)
	t.Cleanup(func() { repo.Close() })
	cluster := &models.Cluster{ID: clusterID, Name: clusterID, Context: "test-ctx", Status: "connected"}
	mockService := &mockClusterServiceWithClient{clusters: []*models.Cluster{cluster}, client: client}
	cfg := &config.Config{AuthMode: "required", AuthJWTSecret: "test-secret-key-minimum-32-characters-long"}
	h := NewHandler(mockService, nil, cfg, nil, nil, nil, nil, nil, nil, repo, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Username: "test-user", Role: role}
			ctx := auth.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	SetupRoutes(api, h)
	return router
}

func toUnstructuredCronJob(cj *batchv1.CronJob) *unstructured.Unstructured {
	cj.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cj)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: obj}
}

func toUnstructuredJob(j *batchv1.Job) *unstructured.Unstructured {
	j.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(j)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: obj}
}

func sampleCronJob(name, namespace string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "backup"},
					Annotations: map[string]string{"owner": "platform-team"},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{Name: "backup", Image: "backup:v1"},
							},
						},
					},
				},
			},
		},
	}
}

func TestHandler_PostCronJobTrigger_Success(t *testing.T) {
	clusterID := "test-cluster"
	cronJob := sampleCronJob("nightly-backup", "default")
	client := newCronJobsTestClient(t, toUnstructuredCronJob(cronJob))
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/nightly-backup/trigger", nil)
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

	// Verify the Job was actually created in the fake cluster, not just echoed
	// back in the response — list jobs and find the one derived from the
	// CronJob's generateName.
	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected exactly 1 Job created, got %d", len(jobList.Items))
	}
	job := jobList.Items[0]
	if genName := job.GetGenerateName(); genName != "nightly-backup-" {
		t.Errorf("expected generateName 'nightly-backup-', got %q", genName)
	}
	if ns := job.GetNamespace(); ns != "default" {
		t.Errorf("expected namespace 'default', got %q", ns)
	}
	// Labels/annotations from jobTemplate.metadata must carry over.
	if job.GetLabels()["app"] != "backup" {
		t.Errorf("expected label app=backup to carry over from jobTemplate, got labels=%v", job.GetLabels())
	}
	if job.GetAnnotations()["owner"] != "platform-team" {
		t.Errorf("expected annotation owner=platform-team to carry over, got annotations=%v", job.GetAnnotations())
	}
	// Spec must be the jobTemplate's spec (restartPolicy, image), not empty.
	containers, found, err := unstructured.NestedSlice(job.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("expected 1 container copied from jobTemplate.spec, found=%v err=%v containers=%v", found, err, containers)
	}
}

func TestHandler_PostCronJobTrigger_NotFound(t *testing.T) {
	clusterID := "test-cluster"
	// Seed a different CronJob so GVR resolution succeeds but the named one 404s.
	client := newCronJobsTestClient(t, toUnstructuredCronJob(sampleCronJob("other-job", "default")))
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/ghost/trigger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 triggering a nonexistent CronJob, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostCronJobTrigger_InvalidName(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredCronJob(sampleCronJob("nightly-backup", "default")))
	router := newCronJobsTestHandler(t, client, clusterID)

	// "/" is not a valid Kubernetes resource name character; validate.Name must reject it
	// before any client call is made.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/bad%2Fname/trigger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("expected 400 or 404 for malformed name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostCronJobTrigger_NoJobTemplate(t *testing.T) {
	clusterID := "test-cluster"
	// A CronJob object missing spec.jobTemplate entirely (hand-built, not
	// via the typed struct, to actually produce that shape).
	broken := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "CronJob",
		"metadata": map[string]interface{}{
			"name":      "broken-cronjob",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"schedule": "*/5 * * * *",
		},
	}}
	client := newCronJobsTestClient(t, broken)
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/broken-cronjob/trigger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for CronJob missing spec.jobTemplate, got %d: %s", rec.Code, rec.Body.String())
	}

	// No Job should have been created.
	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 0 {
		t.Errorf("expected no Job created when jobTemplate is missing, got %d", len(jobList.Items))
	}
}

func TestHandler_PostCronJobTrigger_RBAC_ViewerForbidden(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredCronJob(sampleCronJob("nightly-backup", "default")))
	router := newCronJobsTestHandlerWithAuth(t, client, clusterID, auth.RoleViewer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/nightly-backup/trigger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer role triggering a CronJob, got %d: %s", rec.Code, rec.Body.String())
	}

	jobList, err := client.ListResources(context.Background(), "jobs", "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListResources jobs: %v", err)
	}
	if len(jobList.Items) != 0 {
		t.Errorf("expected no Job created when RBAC denies the viewer, got %d", len(jobList.Items))
	}
}

func TestHandler_PostCronJobTrigger_RBAC_OperatorAllowed(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t, toUnstructuredCronJob(sampleCronJob("nightly-backup", "default")))
	router := newCronJobsTestHandlerWithAuth(t, client, clusterID, auth.RoleOperator)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/nightly-backup/trigger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for operator role triggering a CronJob, got %d: %s", rec.Code, rec.Body.String())
	}
}

func sampleOwnedJob(name, namespace, ownerName string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: ownerName, APIVersion: "batch/v1"},
			},
		},
	}
}

func TestHandler_GetCronJobJobs_FiltersSortsAndLimits(t *testing.T) {
	clusterID := "test-cluster"

	owned1 := toUnstructuredJob(sampleOwnedJob("backup-111", "default", "nightly-backup"))
	setNestedStatusStartTime(owned1, "2026-01-01T00:00:00Z")

	owned2 := toUnstructuredJob(sampleOwnedJob("backup-222", "default", "nightly-backup"))
	setNestedStatusStartTime(owned2, "2026-01-03T00:00:00Z")

	owned3 := toUnstructuredJob(sampleOwnedJob("backup-333", "default", "nightly-backup"))
	setNestedStatusStartTime(owned3, "2026-01-02T00:00:00Z")

	unrelated := toUnstructuredJob(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated-job", Namespace: "default"},
	})

	client := newCronJobsTestClient(t, owned1, owned2, owned3, unrelated)
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/nightly-backup/jobs?limit=2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected limit=2 to cap results at 2, got %d", len(resp.Items))
	}
	// Newest (backup-222, 2026-01-03) must come first.
	firstName, _, _ := unstructured.NestedString(resp.Items[0], "metadata", "name")
	secondName, _, _ := unstructured.NestedString(resp.Items[1], "metadata", "name")
	if firstName != "backup-222" {
		t.Errorf("expected newest job backup-222 first, got %q", firstName)
	}
	if secondName != "backup-333" {
		t.Errorf("expected second-newest job backup-333 second, got %q", secondName)
	}
	for _, item := range resp.Items {
		name, _, _ := unstructured.NestedString(item, "metadata", "name")
		if name == "unrelated-job" {
			t.Error("expected the unowned job to be filtered out")
		}
	}
}

func TestHandler_GetCronJobJobs_NoOwnedJobs_ReturnsEmptyList(t *testing.T) {
	clusterID := "test-cluster"
	unrelated := toUnstructuredJob(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated-job", Namespace: "default"},
	})
	client := newCronJobsTestClient(t, unrelated)
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/"+clusterID+"/resources/cronjobs/default/nightly-backup/jobs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even with no owned jobs, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected empty items list, got %d", len(resp.Items))
	}
}

func TestHandler_GetCronJobJobs_InvalidClusterID(t *testing.T) {
	clusterID := "test-cluster"
	client := newCronJobsTestClient(t)
	router := newCronJobsTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/bad%20cluster!/resources/cronjobs/default/nightly-backup/jobs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("expected 400 or 404 for invalid clusterId, got %d: %s", rec.Code, rec.Body.String())
	}
}

// setNestedStatusStartTime sets status.startTime directly on the
// unstructured object. batchv1.JobStatus's StartTime is a *metav1.Time
// which the typed-struct round trip through DefaultUnstructuredConverter
// renders fine, but building it by hand here keeps the test's intent
// (an arbitrary RFC3339 string for sort ordering) explicit and decoupled
// from metav1.Time's exact JSON marshaling.
func setNestedStatusStartTime(u *unstructured.Unstructured, rfc3339 string) {
	if err := unstructured.SetNestedField(u.Object, rfc3339, "status", "startTime"); err != nil {
		panic(err)
	}
}
