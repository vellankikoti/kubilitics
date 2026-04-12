package kubectl

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ImpactReport describes what would be affected by a delete operation
type ImpactReport struct {
	ResourceType  string
	ResourceName  string
	Namespace     string
	AffectedPods  int
	AffectedSvcs  int
	AffectedRepos int
	Details       []string
}

// AnalyzeDeleteImpact returns what would be affected by a delete operation
func AnalyzeDeleteImpact(kubeconfigPath, ctx, namespace, resourceType, resourceName string) (*ImpactReport, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	if ctx != "" {
		configOverrides.CurrentContext = ctx
	}
	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	report := &ImpactReport{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Namespace:    namespace,
		Details:      make([]string, 0),
	}

	resourceType = strings.ToLower(resourceType)

	switch resourceType {
	case "deployment", "deployments", "deploy":
		return analyzeDeploymentDelete(clientset, namespace, resourceName, report)

	case "statefulset", "statefulsets", "sts":
		return analyzeStatefulSetDelete(clientset, namespace, resourceName, report)

	case "daemonset", "daemonsets", "ds":
		return analyzeDaemonSetDelete(clientset, namespace, resourceName, report)

	case "pod", "pods", "po":
		return analyzePodDelete(clientset, namespace, resourceName, report)

	case "service", "services", "svc":
		return analyzeServiceDelete(clientset, namespace, resourceName, report)

	case "namespace", "ns":
		return analyzeNamespaceDelete(clientset, resourceName, report)

	default:
		report.Details = append(report.Details, fmt.Sprintf("Impact analysis for %s not available", resourceType))
		return report, nil
	}
}

func analyzeDeploymentDelete(clientset kubernetes.Interface, namespace, deployName string, report *ImpactReport) (*ImpactReport, error) {
	deploy, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deployName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err == nil {
		report.AffectedPods = len(podList.Items)
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d pod(s) managed by this deployment", report.AffectedPods))
	}

	svcList, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err == nil {
		affectedSvcs := countServicesWithSelector(svcList, deploy.Labels)
		report.AffectedSvcs = affectedSvcs
		if affectedSvcs > 0 {
			report.Details = append(report.Details, fmt.Sprintf("%d service(s) may be affected", affectedSvcs))
		}
	}

	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err == nil {
		affectedRepos := countOwnedResources(rsList.Items, deploy.UID)
		report.AffectedRepos = affectedRepos
		if affectedRepos > 0 {
			report.Details = append(report.Details, fmt.Sprintf("%d ReplicaSet(s) will be deleted", affectedRepos))
		}
	}

	report.Details = append(report.Details, fmt.Sprintf("Desired replicas: %d", readDesiredReplicas(deploy)))

	return report, nil
}

func analyzeStatefulSetDelete(clientset kubernetes.Interface, namespace, stsName string, report *ImpactReport) (*ImpactReport, error) {
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), stsName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulset: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err == nil {
		report.AffectedPods = len(podList.Items)
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d pod(s) managed by this statefulset", report.AffectedPods))
	}

	pvcList, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{})
	if err == nil {
		affectedPVCs := 0
		for _, pvc := range pvcList.Items {
			for _, owner := range pvc.OwnerReferences {
				if owner.Kind == "StatefulSet" && owner.Name == stsName {
					affectedPVCs++
					report.Details = append(report.Details, fmt.Sprintf("PVC '%s' will be affected", pvc.Name))
				}
			}
		}
		if affectedPVCs > 0 {
			report.Details = append(report.Details, fmt.Sprintf("WARNING: %d PVC(s) may be left behind", affectedPVCs))
		}
	}

	desired := readDesiredStatefulSetReplicas(sts)
	report.Details = append(report.Details, fmt.Sprintf("Desired replicas: %d", desired))

	return report, nil
}

func analyzeDaemonSetDelete(clientset kubernetes.Interface, namespace, dsName string, report *ImpactReport) (*ImpactReport, error) {
	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), dsName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonset: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err == nil {
		report.AffectedPods = len(podList.Items)
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d pod(s) running on nodes", report.AffectedPods))
	}

	report.Details = append(report.Details, "WARNING: DaemonSet pods run on every node - this is a cluster-wide operation")

	return report, nil
}

func analyzePodDelete(clientset kubernetes.Interface, namespace, podName string, report *ImpactReport) (*ImpactReport, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	report.AffectedPods = 1
	report.Details = append(report.Details, fmt.Sprintf("Pod will be terminated immediately"))

	for _, owner := range pod.OwnerReferences {
		report.Details = append(report.Details, fmt.Sprintf("Pod is managed by %s '%s'", owner.Kind, owner.Name))
		report.Details = append(report.Details, fmt.Sprintf("A new pod will likely be created as a replacement"))
	}

	return report, nil
}

func analyzeServiceDelete(clientset kubernetes.Interface, namespace, svcName string, report *ImpactReport) (*ImpactReport, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(context.Background(), svcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	report.AffectedSvcs = 1

	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(context.Background(), svcName, metav1.GetOptions{})
	if err == nil {
		endpointCount := 0
		for _, subset := range endpoints.Subsets {
			endpointCount += len(subset.Addresses)
		}
		report.Details = append(report.Details, fmt.Sprintf("Service has %d endpoint(s)", endpointCount))
	}

	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		report.Details = append(report.Details, "WARNING: LoadBalancer service - external access will be lost")
	}

	if len(svc.Spec.Selector) > 0 {
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: svc.Spec.Selector,
		})
		if err == nil {
			podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: selector.String(),
			})
			if err == nil {
				report.Details = append(report.Details, fmt.Sprintf("Service targets %d pod(s)", len(podList.Items)))
			}
		}
	}

	return report, nil
}

func analyzeNamespaceDelete(clientset kubernetes.Interface, nsName string, report *ImpactReport) (*ImpactReport, error) {
	podList, _ := clientset.CoreV1().Pods(nsName).List(context.Background(), metav1.ListOptions{})
	svcList, _ := clientset.CoreV1().Services(nsName).List(context.Background(), metav1.ListOptions{})
	deployList, _ := clientset.AppsV1().Deployments(nsName).List(context.Background(), metav1.ListOptions{})
	stsList, _ := clientset.AppsV1().StatefulSets(nsName).List(context.Background(), metav1.ListOptions{})
	dsList, _ := clientset.AppsV1().DaemonSets(nsName).List(context.Background(), metav1.ListOptions{})
	pvcList, _ := clientset.CoreV1().PersistentVolumeClaims(nsName).List(context.Background(), metav1.ListOptions{})

	if podList != nil {
		report.AffectedPods = len(podList.Items)
	}
	if svcList != nil {
		report.AffectedSvcs = len(svcList.Items)
	}

	report.Details = append(report.Details, fmt.Sprintf("Will delete %d pod(s)", report.AffectedPods))
	report.Details = append(report.Details, fmt.Sprintf("Will delete %d service(s)", report.AffectedSvcs))
	if deployList != nil {
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d deployment(s)", len(deployList.Items)))
	}
	if stsList != nil {
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d statefulset(s)", len(stsList.Items)))
	}
	if dsList != nil {
		report.Details = append(report.Details, fmt.Sprintf("Will delete %d daemonset(s)", len(dsList.Items)))
	}
	if pvcList != nil && len(pvcList.Items) > 0 {
		report.Details = append(report.Details, fmt.Sprintf("WARNING: Will delete %d PVC(s) and their data", len(pvcList.Items)))
	}

	report.Details = append(report.Details, "CRITICAL: This operation is irreversible")

	return report, nil
}

func countServicesWithSelector(svcList *corev1.ServiceList, labels map[string]string) int {
	count := 0
	for _, svc := range svcList.Items {
		matches := true
		for k, v := range svc.Spec.Selector {
			if labels[k] != v {
				matches = false
				break
			}
		}
		if matches && len(svc.Spec.Selector) > 0 {
			count++
		}
	}
	return count
}

func countOwnedResources(resources []appsv1.ReplicaSet, ownerUID types.UID) int {
	count := 0
	for _, rs := range resources {
		for _, owner := range rs.OwnerReferences {
			if owner.UID == ownerUID {
				count++
				break
			}
		}
	}
	return count
}

func readDesiredStatefulSetReplicas(sts *appsv1.StatefulSet) int32 {
	if sts.Spec.Replicas != nil {
		return *sts.Spec.Replicas
	}
	return 1
}
