/**
 * Maps Kubernetes resource kinds to their official community SVG icon URLs.
 * Vite resolves these imports as asset URLs at build time.
 */
import pod from "./svgs/pod.svg";
import deployment from "./svgs/deployment.svg";
import service from "./svgs/service.svg";
import configmap from "./svgs/configmap.svg";
import secret from "./svgs/secret.svg";
import ingress from "./svgs/ingress.svg";
import namespace from "./svgs/namespace.svg";
import replicaset from "./svgs/replicaset.svg";
import statefulset from "./svgs/statefulset.svg";
import daemonset from "./svgs/daemonset.svg";
import job from "./svgs/job.svg";
import cronjob from "./svgs/cronjob.svg";
import persistentvolume from "./svgs/persistentvolume.svg";
import persistentvolumeclaim from "./svgs/persistentvolumeclaim.svg";
import storageclass from "./svgs/storageclass.svg";
import serviceaccount from "./svgs/serviceaccount.svg";
import role from "./svgs/role.svg";
import clusterrole from "./svgs/clusterrole.svg";
import rolebinding from "./svgs/rolebinding.svg";
import clusterrolebinding from "./svgs/clusterrolebinding.svg";
import networkpolicy from "./svgs/networkpolicy.svg";
import horizontalpodautoscaler from "./svgs/horizontalpodautoscaler.svg";
import endpoints from "./svgs/endpoints.svg";
import limitrange from "./svgs/limitrange.svg";
import node from "./svgs/node.svg";

/** Lowercase kind → SVG asset URL */
const k8sIconMap: Record<string, string> = {
  pod,
  deployment,
  service,
  configmap,
  secret,
  ingress,
  namespace,
  replicaset,
  statefulset,
  daemonset,
  job,
  cronjob,
  persistentvolume,
  persistentvolumeclaim,
  storageclass,
  serviceaccount,
  role,
  clusterrole,
  rolebinding,
  clusterrolebinding,
  networkpolicy,
  horizontalpodautoscaler,
  endpoints,
  limitrange,
  node,
  // Aliases for common shorthand
  pv: persistentvolume,
  pvc: persistentvolumeclaim,
  sc: storageclass,
  sa: serviceaccount,
  hpa: horizontalpodautoscaler,
  rs: replicaset,
  ds: daemonset,
  sts: statefulset,
  svc: service,
  cm: configmap,
  ep: endpoints,
  np: networkpolicy,
  ns: namespace,
  deploy: deployment,
  rb: rolebinding,
  crb: clusterrolebinding,
  cr: clusterrole,
  lr: limitrange,
};

export default k8sIconMap;
