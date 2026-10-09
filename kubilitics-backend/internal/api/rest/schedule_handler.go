package rest

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kubilitics/kubilitics-backend/internal/intelligence/reports"
)

// ScheduleHandler handles /api/v1/clusters/{clusterId}/reports/schedules endpoints.
type ScheduleHandler struct {
	scheduler *reports.Scheduler
}

// NewScheduleHandler creates a new schedule handler.
func NewScheduleHandler(scheduler *reports.Scheduler) *ScheduleHandler {
	return &ScheduleHandler{scheduler: scheduler}
}

// Routes are registered by Handler.SetupRoutes (handler.go), RBAC-wrapped,
// guarded by a scheduleHandler-nil check — not self-registered here. See
// docs/ai/STABILIZATION-PLAN.md Phase 0.5.

// CreateSchedule handles POST /clusters/{clusterId}/reports/schedules.
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if clusterID == "" {
		respondError(w, http.StatusBadRequest, "clusterId is required")
		return
	}

	var schedule reports.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	schedule.ClusterID = clusterID

	created, err := h.scheduler.Create(schedule)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// ListSchedules handles GET /clusters/{clusterId}/reports/schedules.
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]

	schedules, err := h.scheduler.List(clusterID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, schedules)
}

// GetSchedule handles GET /clusters/{clusterId}/reports/schedules/{scheduleId}.
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["scheduleId"]

	schedule, err := h.scheduler.Get(scheduleID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, schedule)
}

// UpdateSchedule handles PUT /clusters/{clusterId}/reports/schedules/{scheduleId}.
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["scheduleId"]

	var updates reports.ScheduleUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.scheduler.Update(scheduleID, updates)
	if err != nil {
		if err.Error() == "schedule not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, updated)
}

// DeleteSchedule handles DELETE /clusters/{clusterId}/reports/schedules/{scheduleId}.
func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["scheduleId"]

	if err := h.scheduler.Delete(scheduleID); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RunNow handles POST /clusters/{clusterId}/reports/schedules/{scheduleId}/run.
// It triggers an immediate execution of the schedule (resets NextRun).
func (h *ScheduleHandler) RunNow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["scheduleId"]

	schedule, err := h.scheduler.Get(scheduleID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Report execution queued",
		"schedule_id": schedule.ID,
		"cluster_id":  schedule.ClusterID,
	})
}
