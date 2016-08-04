package caicloudprocessors

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/golang/glog"
	kubeconfig "k8s.io/heapster/common/kubernetes"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
	kubeapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

type PodBasedEnricher struct {
	podLister *cache.StoreToPodLister
}

func (p *PodBasedEnricher) Name() string {
	return "pod_based_enricher"
}

func (p *PodBasedEnricher) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	podsOrContainers := make(map[string]*core.MetricSet, len(batch.MetricSets))

	for key, metricSet := range batch.MetricSets {
		namespaceName := metricSet.Labels[core.LabelNamespaceName.Key]
		podName := metricSet.Labels[core.LabelPodName.Key]

		pod, err := p.getPod(namespaceName, podName)
		if err != nil {
			glog.V(3).Infof("Failed to get pod %s from cache: %v", core.PodKey(namespaceName, podName), err)
			continue
		}
		switch metricSet.Labels[core.LabelMetricSetType.Key] {
		case core.MetricSetTypePod:
			changePodInfo(key, metricSet, pod, batch, podsOrContainers)
		case core.MetricSetTypePodContainer:
			changeContainerInfo(key, metricSet, pod, batch, podsOrContainers)
		}
	}
	for k, v := range podsOrContainers {
		batch.MetricSets[k] = v
	}
	return batch, nil
}

func (p *PodBasedEnricher) getPod(namespaceName, podName string) (*kubeapi.Pod, error) {
	o, exists, err := p.podLister.Get(
		&kubeapi.Pod{
			ObjectMeta: kubeapi.ObjectMeta{
				Namespace: namespaceName,
				Name:      podName,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if !exists || o == nil {
		return nil, fmt.Errorf("cannot find pod definition")
	}
	pod, ok := o.(*kubeapi.Pod)
	if !ok {
		return nil, fmt.Errorf("cache contains wrong type")
	}
	return pod, nil
}

var containerLabelsToPopulate = []core.LabelDescriptor{
	core.LabelPodName,
	core.LabelNamespaceName,
	core.LabelHostname,
	caicloudcore.LabelAppName,
	core.LabelContainerName,
	core.LabelContainerBaseImage,
}

func changeContainerInfo(key string, containerMs *core.MetricSet, pod *kubeapi.Pod, batch *core.DataBatch, newMs map[string]*core.MetricSet) {

	appName := getApplicationName(pod)

	for _, container := range pod.Spec.Containers {
		if key == core.PodContainerKey(pod.Namespace, pod.Name, container.Name) {
			updateContainerResourcesAndLimits(containerMs, &container)
			if _, ok := containerMs.Labels[core.LabelContainerBaseImage.Key]; !ok {
				containerMs.Labels[core.LabelContainerBaseImage.Key] = container.Image
			}
			break
		}
	}

	containerMs.Labels[core.LabelPodId.Key] = string(pod.UID)
	containerMs.Labels[caicloudcore.LabelAppName.Key] = appName

	namespaceName := containerMs.Labels[core.LabelNamespaceName.Key]
	podName := containerMs.Labels[core.LabelPodName.Key]

	podKey := core.PodKey(namespaceName, podName)
	_, oldfound := batch.MetricSets[podKey]
	if !oldfound {
		_, newfound := newMs[podKey]
		if !newfound {
			glog.V(2).Infof("Pod %s not found, creating a stub", podKey)
			podMs := &core.MetricSet{
				MetricValues: make(map[string]core.MetricValue),
				Labels: map[string]string{
					core.LabelMetricSetType.Key:   core.MetricSetTypePod,
					core.LabelNamespaceName.Key:   namespaceName,
					core.LabelPodNamespace.Key:    namespaceName,
					caicloudcore.LabelAppName.Key: appName,
					core.LabelPodName.Key:         podName,
					core.LabelNodename.Key:        containerMs.Labels[core.LabelNodename.Key],
					core.LabelHostname.Key:        containerMs.Labels[core.LabelHostname.Key],
					core.LabelHostID.Key:          containerMs.Labels[core.LabelHostID.Key],
				},
			}
			newMs[podKey] = podMs
			changePodInfo(podKey, podMs, pod, batch, newMs)
		}
	}
}

func changePodInfo(key string, podMs *core.MetricSet, pod *kubeapi.Pod, batch *core.DataBatch, newMs map[string]*core.MetricSet) {

	appName := getApplicationName(pod)

	podMs.Labels[core.LabelPodId.Key] = string(pod.UID)
	podMs.Labels[caicloudcore.LabelAppName.Key] = appName

	// Add cpu/mem requests and limits to containers
	for _, container := range pod.Spec.Containers {
		containerKey := core.PodContainerKey(pod.Namespace, pod.Name, container.Name)
		if _, found := batch.MetricSets[containerKey]; !found {
			if _, found := newMs[containerKey]; !found {
				glog.V(2).Infof("Container %s not found, creating a stub", containerKey)
				containerMs := &core.MetricSet{
					MetricValues: make(map[string]core.MetricValue),
					Labels: map[string]string{
						core.LabelMetricSetType.Key:      core.MetricSetTypePodContainer,
						core.LabelNamespaceName.Key:      pod.Namespace,
						core.LabelPodNamespace.Key:       pod.Namespace,
						caicloudcore.LabelAppName.Key:    appName,
						core.LabelPodName.Key:            pod.Name,
						core.LabelContainerName.Key:      container.Name,
						core.LabelContainerBaseImage.Key: container.Image,
						core.LabelPodId.Key:              string(pod.UID),
						core.LabelNodename.Key:           podMs.Labels[core.LabelNodename.Key],
						core.LabelHostname.Key:           podMs.Labels[core.LabelHostname.Key],
						core.LabelHostID.Key:             podMs.Labels[core.LabelHostID.Key],
					},
				}
				updateContainerResourcesAndLimits(containerMs, &container)
				newMs[containerKey] = containerMs
			}
		}
	}
}

// use Label firstly
// use created by secondly
// just return pod name finally
// TODO(liubog2008): maybe label name will be same as others' default name
func getApplicationName(pod *kubeapi.Pod) string {
	const appLableKey = "kubernetes-admin.caicloud.io/application"
	const createdByAnnotation = "kubernetes.io/created-by"
	if name, ok := pod.Labels[appLableKey]; ok {
		return name
	} else if jsonRef, ok := pod.Annotations[createdByAnnotation]; ok {
		ref := kubeapi.SerializedReference{}
		if err := json.Unmarshal([]byte(jsonRef), &ref); err != nil {
			glog.Errorf("can't get applicaton name: %v", err)
		} else {
			name := ref.Reference.Kind + ":" + ref.Reference.Name
			return name
		}
	}

	return pod.Name
}

func updateContainerResourcesAndLimits(metricSet *core.MetricSet, container *kubeapi.Container) {
	requests := container.Resources.Requests
	if val, found := requests[kubeapi.ResourceCPU]; found {
		metricSet.MetricValues[core.MetricCpuRequest.Name] = intValue(val.MilliValue())
	} else {
		metricSet.MetricValues[core.MetricCpuRequest.Name] = intValue(caicloudcore.UNLIMIT)
	}
	if val, found := requests[kubeapi.ResourceMemory]; found {
		metricSet.MetricValues[core.MetricMemoryRequest.Name] = intValue(val.Value())
	} else {
		metricSet.MetricValues[core.MetricMemoryRequest.Name] = intValue(caicloudcore.UNLIMIT)
	}

	limits := container.Resources.Limits
	if val, found := limits[kubeapi.ResourceCPU]; found {
		metricSet.MetricValues[core.MetricCpuLimit.Name] = intValue(val.MilliValue())
	} else {
		metricSet.MetricValues[core.MetricCpuLimit.Name] = intValue(caicloudcore.UNLIMIT)
	}
	if val, found := limits[kubeapi.ResourceMemory]; found {
		metricSet.MetricValues[core.MetricMemoryLimit.Name] = intValue(val.Value())
	} else {
		metricSet.MetricValues[core.MetricMemoryLimit.Name] = intValue(caicloudcore.UNLIMIT)
	}
}

func NewPodBasedEnricher(url *url.URL) (*PodBasedEnricher, error) {
	kubeConfig, err := kubeconfig.GetKubeClientConfig(url)
	if err != nil {
		return nil, err
	}
	kubeClient := kubeclient.NewOrDie(kubeConfig)

	lw := cache.NewListWatchFromClient(kubeClient, "pods", kubeapi.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	podLister := &cache.StoreToPodLister{Indexer: store}
	reflector := cache.NewReflector(lw, &kubeapi.Pod{}, store, time.Hour)
	reflector.Run()

	return &PodBasedEnricher{
		podLister: podLister,
	}, nil
}
