package output

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func StatusColor(resourceKind, statusString string) lipgloss.Style {
	return StatusStyle(resourceKind, statusString)
}

func StatusIcon(status string) string {
	status = strings.ToLower(status)

	switch status {
	case "running", "ready", "active", "bound", "succeeded":
		return "✓"
	case "pending", "terminating", "creating":
		return "⏳"
	case "failed", "crashloopbackoff", "error", "errorimagepull", "imagepullbackoff", "lost":
		return "✗"
	case "notready", "noschedule", "unknown":
		return "—"
	default:
		return "?"
	}
}

func PodPhaseStyle(phase string) lipgloss.Style {
	return StatusStyle("pod", phase)
}

func PodPhaseIcon(phase string) string {
	phase = strings.ToLower(phase)

	switch phase {
	case "running":
		return "✓"
	case "pending":
		return "⏳"
	case "succeeded":
		return "✓"
	case "failed":
		return "✗"
	case "unknown":
		return "?"
	default:
		return "—"
	}
}

func NodeConditionStyle(condition string) lipgloss.Style {
	return StatusStyle("node", condition)
}

func NodeReadyIcon(ready bool) string {
	if ready {
		return "✓"
	}
	return "✗"
}

func DeploymentReadyStyle(ready, desired int) lipgloss.Style {
	theme := GetTheme()
	if ready == desired && ready > 0 {
		return theme.StatusReady
	}
	if ready == 0 {
		return theme.StatusError
	}
	return theme.StatusPending
}

func DeploymentReadyIcon(ready, desired int) string {
	if ready == desired && ready > 0 {
		return "✓"
	}
	if ready == 0 {
		return "✗"
	}
	return "⏳"
}

func PVCPhaseStyle(phase string) lipgloss.Style {
	return StatusStyle("pvc", phase)
}

func PVCPhaseIcon(phase string) string {
	phase = strings.ToLower(phase)

	switch phase {
	case "bound":
		return "✓"
	case "pending":
		return "⏳"
	case "failed", "lost":
		return "✗"
	default:
		return "?"
	}
}

func JobStatusStyle(status string) lipgloss.Style {
	status = strings.ToLower(status)

	theme := GetTheme()
	switch status {
	case "complete":
		return theme.StatusReady
	case "failed":
		return theme.StatusError
	case "active":
		return theme.StatusPending
	default:
		return theme.StatusUnknown
	}
}

func JobStatusIcon(status string) string {
	status = strings.ToLower(status)

	switch status {
	case "complete":
		return "✓"
	case "failed":
		return "✗"
	case "active":
		return "⏳"
	default:
		return "?"
	}
}

func StatefulSetReadyStyle(ready, desired int) lipgloss.Style {
	return DeploymentReadyStyle(ready, desired)
}

func StatefulSetReadyIcon(ready, desired int) string {
	return DeploymentReadyIcon(ready, desired)
}

func DaemonSetReadyStyle(ready, desired int) lipgloss.Style {
	return DeploymentReadyStyle(ready, desired)
}

func DaemonSetReadyIcon(ready, desired int) string {
	return DeploymentReadyIcon(ready, desired)
}

func ServiceTypeStyle(serviceType string) lipgloss.Style {
	return GetTheme().Primary
}

func IngressStatusStyle(status string) lipgloss.Style {
	status = strings.ToLower(status)

	theme := GetTheme()
	if status == "" {
		return theme.StatusPending
	}
	return theme.StatusReady
}

func EventTypeStyle(eventType string) lipgloss.Style {
	eventType = strings.ToLower(eventType)

	theme := GetTheme()
	switch eventType {
	case "normal":
		return theme.Info
	case "warning":
		return theme.Warning
	default:
		return theme.Primary
	}
}
