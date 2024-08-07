package utils

import v1 "k8s.io/api/core/v1"

func PodCpuLimit(pod v1.Pod) int64 {
	var cpu int64
	for _, container := range pod.Spec.Containers {
		cpu += container.Resources.Limits.Cpu().Value()
	}
	return cpu
}

func PodCpuRequest(pod v1.Pod) int64 {
	var cpu int64
	for _, container := range pod.Spec.Containers {
		cpu += container.Resources.Requests.Cpu().Value()
	}
	return cpu
}

func PodMemLimit(pod v1.Pod) int64 {
	var mem int64
	for _, container := range pod.Spec.Containers {
		mem += container.Resources.Limits.Memory().Value()
	}
	return mem
}

func PodMemRequest(pod v1.Pod) int64 {
	var mem int64
	for _, container := range pod.Spec.Containers {
		mem += container.Resources.Requests.Memory().Value()
	}
	return mem
}

func PodGpuNvidia(pod v1.Pod) int64 {
	var gpuNum int64
	for _, container := range pod.Spec.Containers {
		gpuNum += container.Resources.Limits.Name("nvidia.com/gpu", "0").Value()
	}
	return gpuNum
}
