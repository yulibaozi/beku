package beku

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
)

func setTolerations(podTemp *v1.PodTemplateSpec, toleration v1.Toleration) {
	if len(podTemp.Spec.Tolerations) <= 0 {
		podTemp.Spec.Tolerations = []v1.Toleration{toleration}
		return
	}
	podTemp.Spec.Tolerations = append(podTemp.Spec.Tolerations, toleration)
}

func setImagePullSecrets(podTemp *v1.PodTemplateSpec, secretName string) {
	if len(podTemp.Spec.ImagePullSecrets) <= 0 {
		podTemp.Spec.ImagePullSecrets = []v1.LocalObjectReference{{Name: secretName}}
		return
	}
	podTemp.Spec.ImagePullSecrets = append(podTemp.Spec.ImagePullSecrets, v1.LocalObjectReference{Name: secretName})
}

func setNodeAffinity(podTemp *v1.PodTemplateSpec, nodeAffinity *v1.NodeAffinity) error {
	if nodeAffinity == nil {
		return errors.New("setNodeAffinity err, NodeAffinity is not allowed to be empty")
	}
	if podTemp.Spec.Affinity == nil {
		podTemp.Spec.Affinity = &v1.Affinity{NodeAffinity: nodeAffinity}
		return nil
	}
	podTemp.Spec.Affinity.NodeAffinity = nodeAffinity
	return nil
}

// delNodeAffinity delete node affinity
func delNodeAffinity(podTemp *v1.PodTemplateSpec, keys []string) error {
	if podTemp.Spec.Affinity != nil {
		if podTemp.Spec.Affinity.NodeAffinity != nil {
			if podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				if len(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
						delNodeSelectorTerms(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, keys)

				}
			}
			// PreferredDuringSchedulingIgnoredDuringExecution
			if len(podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
				podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
					delPreferkeys(podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, keys)
			}
		}
	}
	return nil
}

var (
	tempTerm = v1.NodeSelectorTerm{}
)

func delPreferkeys(terms []v1.PreferredSchedulingTerm, keys []string) []v1.PreferredSchedulingTerm {
	targents := terms[:0]
	for i := range terms {
		if reflect.DeepEqual(terms[i], tempTerm) {
			continue
		}
		reqs := delMatchExpressions(terms[i].Preference.MatchExpressions, keys)
		if len(terms[i].Preference.MatchFields) <= 0 && len(reqs) <= 0 {
			continue
		}
		targents = append(targents, v1.PreferredSchedulingTerm{
			Weight: terms[i].Weight,
			Preference: v1.NodeSelectorTerm{
				MatchExpressions: reqs,
				MatchFields:      terms[i].Preference.MatchFields,
			}})
	}
	if len(targents) <= 0 {
		return nil
	}
	return targents
}

func delNodeSelectorTerms(terms []v1.NodeSelectorTerm, keys []string) []v1.NodeSelectorTerm {
	if len(terms) <= 0 {
		return nil
	}
	targents := []v1.NodeSelectorTerm{}
	for i := range terms {
		if len(terms[i].MatchExpressions) > 0 {
			reqs := delMatchExpressions(terms[i].MatchExpressions, keys)
			if len(reqs) > 0 {
				targents = append(targents, v1.NodeSelectorTerm{MatchExpressions: reqs, MatchFields: terms[i].MatchFields})
				continue
			}
			if len(terms[i].MatchFields) <= 0 {
				continue
			}
			targents = append(targents, v1.NodeSelectorTerm{MatchFields: terms[i].MatchFields})

		}
	}
	if len(targents) > 0 {
		return targents
	}
	return nil
}

func delMatchExpressions(reqs []v1.NodeSelectorRequirement, keys []string) []v1.NodeSelectorRequirement {
	targents := []v1.NodeSelectorRequirement{}
	for i := range reqs {
		if !requirementKeyExist(keys, reqs[i].Key) {
			targents = append(targents, reqs[i])
		}
	}
	return targents
}

func requirementKeyExist(keys []string, key string) bool {
	for i := range keys {
		if keys[i] == key {
			return true
		}
	}
	return false
}

func setRequiredORNodeAffinity(podTemp *v1.PodTemplateSpec, nsRequirement v1.NodeSelectorRequirement) (err error) {
	term := v1.NodeSelectorTerm{MatchExpressions: []v1.NodeSelectorRequirement{nsRequirement}}
	if podTemp.Spec.Affinity != nil {
		if podTemp.Spec.Affinity.NodeAffinity != nil {
			if podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				if len(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
						append(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, term)
					return
				}
				podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []v1.NodeSelectorTerm{term}
				return
			}
			podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}}
			return
		}
		podTemp.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}},
		}
		return
	}
	podTemp.Spec.Affinity = &v1.Affinity{NodeAffinity: &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}}}}
	return
}

func setRequiredAndNodeAffinity(podTemp *v1.PodTemplateSpec, nsRequirement v1.NodeSelectorRequirement) (err error) {
	term := v1.NodeSelectorTerm{MatchExpressions: []v1.NodeSelectorRequirement{nsRequirement}}
	if podTemp.Spec.Affinity != nil {
		if podTemp.Spec.Affinity.NodeAffinity != nil {
			if podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				if len(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					if len(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions) > 0 {
						podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
							append(podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, nsRequirement)
						return
					}
					podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
						[]v1.NodeSelectorRequirement{nsRequirement}
					return
				}
				podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []v1.NodeSelectorTerm{term}
				return
			}
			podTemp.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}}
			return
		}
		podTemp.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}}}
		return
	}
	podTemp.Spec.Affinity = &v1.Affinity{NodeAffinity: &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{term}}}}
	return
}

func setPreferredNodeAffinity(podTemp *v1.PodTemplateSpec, nsRequirement v1.NodeSelectorRequirement, weight int32) (err error) {
	term := v1.PreferredSchedulingTerm{
		Weight:     weight,
		Preference: v1.NodeSelectorTerm{MatchExpressions: []v1.NodeSelectorRequirement{nsRequirement}},
	}
	if podTemp.Spec.Affinity != nil {
		if podTemp.Spec.Affinity.NodeAffinity != nil {
			if len(podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
				podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
					append(podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
				return
			}
			podTemp.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []v1.PreferredSchedulingTerm{term}
			return
		}
		podTemp.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{term}}
		return
	}

	podTemp.Spec.Affinity = &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{term}},
	}
	return
}

// setContainer set container
func setContainer(podTemp *v1.PodTemplateSpec, name, image string, containerPort int32) error {
	// This must be a valid port number, 0 < x < 65536.
	if containerPort <= 0 || containerPort >= 65536 {
		return errors.New("SetContainer err, container Port range: 0 < containerPort < 65536")
	}
	if !verifyString(image) {
		return errors.New("SetContainer err, image is not allowed to be empty")

	}
	port := v1.ContainerPort{ContainerPort: containerPort}
	container := v1.Container{
		Name:  name,
		Image: image,
		Ports: []v1.ContainerPort{port},
	}
	containersLen := len(podTemp.Spec.Containers)
	if containersLen < 1 {
		podTemp.Spec.Containers = []v1.Container{container}
		return nil
	}
	for index := 0; index < containersLen; index++ {
		img := strings.TrimSpace(podTemp.Spec.Containers[index].Image)
		if img == "" || len(img) <= 0 {
			podTemp.Spec.Containers[index].Name = name
			podTemp.Spec.Containers[index].Image = image
			podTemp.Spec.Containers[index].Ports = []v1.ContainerPort{port}
			return nil
		}
	}
	podTemp.Spec.Containers = append(podTemp.Spec.Containers, container)
	return nil
}

func setResourceLimit(podTemp *v1.PodTemplateSpec, limits map[ResourceName]string) error {
	data, err := ResourceMapsToK8s(limits)
	if err != nil {
		return fmt.Errorf("SetResourceLimit err:%v", err)
	}
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Resources: v1.ResourceRequirements{Limits: data}}}
		return nil
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Resources.Limits == nil {
			podTemp.Spec.Containers[index].Resources.Limits = data
		}
	}
	return nil
}

func setResourceRequests(podTemp *v1.PodTemplateSpec, requests map[ResourceName]string) error {
	data, err := ResourceMapsToK8s(requests)
	if err != nil {
		return fmt.Errorf("SetResourceLimit err:%v", err)
	}
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Resources: v1.ResourceRequirements{Requests: data}}}
		return nil
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Resources.Requests == nil {
			podTemp.Spec.Containers[index].Resources.Requests = data
		}
	}
	return nil
}

func setPreStopExec(podTemp *v1.PodTemplateSpec, command []string) {
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Lifecycle: &v1.Lifecycle{PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: command}}}}}
		return
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Lifecycle == nil {
			podTemp.Spec.Containers[index].Lifecycle = &v1.Lifecycle{PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: command}}}
			return
		}
		if podTemp.Spec.Containers[index].Lifecycle.PreStop == nil {
			podTemp.Spec.Containers[index].Lifecycle.PreStop = &v1.Handler{Exec: &v1.ExecAction{Command: command}}
			return
		}
		continue
		// podTemp.Spec.Containers[index].Lifecycle.PreStop.Exec = &v1.ExecAction{Command: command}
		// return
	}
}

func setPreStopHTTP(podTemp *v1.PodTemplateSpec, scheme URIScheme, host string, port int, path string, headers ...map[string]string) {
	httpAction := &v1.HTTPGetAction{Path: path, Port: FromInt(port), HTTPHeaders: mapsToHeaders(headers), Scheme: v1.URIScheme(scheme)}
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Lifecycle: &v1.Lifecycle{PreStop: &v1.Handler{HTTPGet: httpAction}}}}
		return
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Lifecycle == nil {
			podTemp.Spec.Containers[index].Lifecycle = &v1.Lifecycle{PreStop: &v1.Handler{HTTPGet: httpAction}}
			return
		}
		if podTemp.Spec.Containers[index].Lifecycle.PreStop == nil {
			podTemp.Spec.Containers[index].Lifecycle.PreStop = &v1.Handler{HTTPGet: httpAction}
			return
		}
		continue
		// podTemp.Spec.Containers[index].Lifecycle.PreStop.HTTPGet = httpAction
		// return
	}
}

func setPostStartExec(podTemp *v1.PodTemplateSpec, command []string) {
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Lifecycle: &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: command}}}}}
		return
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Lifecycle == nil {
			podTemp.Spec.Containers[index].Lifecycle = &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: command}}}
			return
		}
		if podTemp.Spec.Containers[index].Lifecycle.PostStart == nil {
			podTemp.Spec.Containers[index].Lifecycle.PostStart = &v1.Handler{Exec: &v1.ExecAction{Command: command}}
			return
		}
		continue
	}
}

func setPostStartHTTP(podTemp *v1.PodTemplateSpec, scheme URIScheme, host string, port int, path string, headers ...map[string]string) {
	httpAction := &v1.HTTPGetAction{Path: path, Port: FromInt(port), HTTPHeaders: mapsToHeaders(headers), Scheme: v1.URIScheme(scheme)}
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Lifecycle: &v1.Lifecycle{PostStart: &v1.Handler{HTTPGet: httpAction}}}}
		return
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Lifecycle == nil {
			podTemp.Spec.Containers[index].Lifecycle = &v1.Lifecycle{PostStart: &v1.Handler{HTTPGet: httpAction}}
			return
		}
		if podTemp.Spec.Containers[index].Lifecycle.PostStart == nil {
			podTemp.Spec.Containers[index].Lifecycle.PostStart = &v1.Handler{HTTPGet: httpAction}
			return
		}
		continue
		// podTemp.Spec.Containers[index].Lifecycle.PostStart.HTTPGet = httpAction
		// return
	}

}

func setPodPriorityClass(podTemp *v1.PodTemplateSpec, priorityClassName string) error {
	if !verifyString(priorityClassName) {
		return errors.New("Set Pod PriorityClass err,priorityClassName is not allowed to be empty")
	}
	podTemp.Spec.PriorityClassName = priorityClassName
	return nil

}

func setEnvs(podTemp *v1.PodTemplateSpec, envMap map[string]string) error {
	envs, err := mapToEnvs(envMap)
	if err != nil {
		return err
	}
	containerLen := len(podTemp.Spec.Containers)
	if containerLen < 1 {
		podTemp.Spec.Containers = []v1.Container{{Env: envs}}
		return nil
	}
	for index := 0; index < containerLen; index++ {
		if podTemp.Spec.Containers[index].Env == nil {
			podTemp.Spec.Containers[index].Env = envs
		}
	}
	return nil
}

func setPVCMounts(podTemp *v1.PodTemplateSpec, volumeName, mountPath string) error {
	volumeMount := v1.VolumeMount{Name: volumeName, MountPath: mountPath}
	if len(podTemp.Spec.Containers) <= 0 {
		podTemp.Spec.Containers = append(podTemp.Spec.Containers, v1.Container{
			VolumeMounts: []v1.VolumeMount{volumeMount},
		})
		return nil
	}
	//only mount first container and first container can mount many data source.
	if len(podTemp.Spec.Containers[0].VolumeMounts) <= 0 {
		podTemp.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{volumeMount}
		return nil
	}
	podTemp.Spec.Containers[0].VolumeMounts = append(podTemp.Spec.Containers[0].VolumeMounts, volumeMount)
	return nil
}

func setPVClaim(podTemp *v1.PodTemplateSpec, volumeName, claimName string) error {
	volume := v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
				ReadOnly:  false,
			},
		},
	}
	if len(podTemp.Spec.Volumes) <= 0 {
		podTemp.Spec.Volumes = []v1.Volume{volume}
		return nil
	}
	podTemp.Spec.Volumes = append(podTemp.Spec.Volumes, volume)
	return nil
}

func setLiveness(podTemp *v1.PodTemplateSpec, probe *v1.Probe) error {
	if len(podTemp.Spec.Containers) <= 0 {
		podTemp.Spec.Containers = []v1.Container{{LivenessProbe: probe}}
		return nil
	}
	for index := range podTemp.Spec.Containers {
		if podTemp.Spec.Containers[index].LivenessProbe == nil {
			podTemp.Spec.Containers[index].LivenessProbe = probe
			return nil
		}
		continue
	}
	return nil
}
func setReadness(podTemp *v1.PodTemplateSpec, probe *v1.Probe) error {
	if len(podTemp.Spec.Containers) <= 0 {
		podTemp.Spec.Containers = []v1.Container{{ReadinessProbe: probe}}
		return nil
	}
	for index := range podTemp.Spec.Containers {
		if podTemp.Spec.Containers[index].ReadinessProbe == nil {
			podTemp.Spec.Containers[index].ReadinessProbe = probe
			return nil
		}
		continue
	}
	return nil
}

var supportedQoSComputeResources = sets.NewString(string(ResourceCPU), string(ResourceMemory))

// QOSList is a set of (resource name, QoS class) pairs.
type QOSList map[v1.ResourceName]v1.PodQOSClass

func isSupportedQoSComputeResource(name v1.ResourceName) bool {
	return supportedQoSComputeResources.Has(string(name))
}

// GetPodQOS returns the QoS class of a pod.
// A pod is besteffort if none of its containers have specified any requests or limits.
// A pod is guaranteed only when requests and limits are specified for all the containers and they are equal.
// A pod is burstable if limits and requests do not match across all containers.
func GetPodQOS(pod v1.PodSpec) v1.PodQOSClass {
	requests := v1.ResourceList{}
	limits := v1.ResourceList{}
	zeroQuantity := resource.MustParse("0")
	isGuaranteed := true
	for _, container := range pod.Containers {
		// process requests
		for name, quantity := range container.Resources.Requests {
			if !isSupportedQoSComputeResource(name) {
				continue
			}
			if quantity.Cmp(zeroQuantity) == 1 {
				delta := quantity.Copy()
				if _, exists := requests[name]; !exists {
					requests[name] = *delta
				} else {
					delta.Add(requests[name])
					requests[name] = *delta
				}
			}
		}
		// process limits
		qosLimitsFound := sets.NewString()
		for name, quantity := range container.Resources.Limits {
			if !isSupportedQoSComputeResource(name) {
				continue
			}
			if quantity.Cmp(zeroQuantity) == 1 {
				qosLimitsFound.Insert(string(name))
				delta := quantity.Copy()
				if _, exists := limits[name]; !exists {
					limits[name] = *delta
				} else {
					delta.Add(limits[name])
					limits[name] = *delta
				}
			}
		}

		if !qosLimitsFound.HasAll(string(v1.ResourceMemory), string(v1.ResourceCPU)) {
			isGuaranteed = false
		}
	}
	if len(requests) == 0 && len(limits) == 0 {
		return v1.PodQOSBestEffort
	}
	// Check is requests match limits for all resources.
	if isGuaranteed {
		for name, req := range requests {
			if lim, exists := limits[name]; !exists || lim.Cmp(req) != 0 {
				isGuaranteed = false
				break
			}
		}
	}
	if isGuaranteed &&
		len(requests) == len(limits) {
		return v1.PodQOSGuaranteed
	}
	return v1.PodQOSBurstable
}

func autoSetQos(targetQos, presentQos string, pod *v1.PodSpec) error {
	if qosRanks[presentQos] >= qosRanks[targetQos] {
		return nil
	}
	if qosRanks[targetQos] == GuaranteedRank &&
		qosRanks[presentQos] == BestEffortRank {
		if len(defaultLimit()) == 2 && len(defaultRequest()) == 2 {
			if reflect.DeepEqual(defaultLimit(), defaultRequest()) {
				containers := len(pod.Containers)
				requests, _ := ResourceMapsToK8s(defaultRequest())
				for index := 0; index < containers; index++ {
					pod.Containers[index].Resources.Limits = requests
					pod.Containers[index].Resources.Requests = requests
				}
				return nil
			}
			return fmt.Errorf("set QOS rank:%s failed,because,Because the default addition is not satisfied,notice:%s", targetQos, qosNotices[targetQos])
		}
		return fmt.Errorf("set QOS rank:%s failed,because,Because the default addition is not satisfied,you can call func RegisterResourceLimit() and RegisterResourceRequest() register default resource limits and requests", targetQos)
	}

	//If what you expect is Burstable
	if len(defaultRequest()) > 0 {
		//set container of Pod resoource requests value.
		containers := len(pod.Containers)
		requests, _ := ResourceMapsToK8s(defaultRequest())
		for index := 0; index < containers; index++ {
			pod.Containers[index].Resources.Requests = requests
		}
		return nil
	}
	return errors.New("set Qos Rank failed,you can call func RegisterResourceLimit() and RegisterResourceRequest() register default resource limits and requests")
}

func qosCheck(qosClass string, podTem v1.PodSpec) (string, error) {
	qosClass = strings.TrimSpace(qosClass)
	if qosClass == "" || qosClass == "BestEffort" {
		return "BestEffort", nil
	}
	//Kubernetes evaluate qos grade
	evaQos := string(GetPodQOS(podTem))
	if qosClass == evaQos {
		return evaQos, nil
	}
	return evaQos, fmt.Errorf("qos check failed, notice:%s", qosNotices[qosClass])
}

// setQosMap set Pod Qos
func setQosMap(dec map[string]string, qosClass string, autoSet ...bool) map[string]string {
	var (
		auto = "false"
	)
	if len(autoSet) > 0 && autoSet[0] {
		auto = "true"

	}
	if dec == nil {
		dec = map[string]string{qosKey: qosClass, autoQosKey: auto}
		return dec
	}
	dec[qosKey] = qosClass
	dec[autoQosKey] = auto
	return dec
}
