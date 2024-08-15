/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"cmp"
	"context"
	"fmt"
	eciv1 "github.com/capitalonline/eci-manager/api/v1"
	"github.com/capitalonline/eci-manager/internal/constants"
	"github.com/capitalonline/eci-manager/internal/utils"
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"slices"
	"time"
)

// PodRecordReconciler reconciles a PodRecord object
type PodRecordReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ExcludeRules []ExcludeRules
	CustomerID   string
}

// +kubebuilder:rbac:groups=eci.eci.cds,resources=podrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eci.eci.cds,resources=podrecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=eci.eci.cds,resources=podrecords/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodRecord object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *PodRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	pod := v1.Pod{}
	record := eciv1.PodRecord{}

	// pod 被删除或者不存在
	if err := r.Client.Get(ctx, req.NamespacedName, &pod); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get pod, error: %v", err)
	}

	if pod.Labels != nil {
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      pod.Labels[constants.LabelPodEciRecord],
		}, &record); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get podrecord, error: %v", err)
		}
	}
	status := podStatus(pod)

	switch status {
	case constants.PodStatusRunning:
		if record.Name != "" {
			return ctrl.Result{}, nil
		}
		newRecord, err := r.newRecord(pod)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create eci-record, error: %v", err)
		}
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		patch := client.StrategicMergeFrom(pod.DeepCopy())
		pod.Labels[constants.LabelPodEciRecord] = newRecord.Name
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return client.IgnoreNotFound(r.Patch(ctx, &pod, patch))
		}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update pod to add label, error: %v", err)
		}
		if err = r.Create(ctx, newRecord); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create eci-record to cluster, error: %v", err)
		}
		klog.Infof("record %s created success", pod.Name)
	default:
		// pod被删除
		if pod.Status.Phase == "" {
			records := eciv1.PodRecordList{}
			if err := r.List(ctx, &records, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(constants.FieldSelectorPodName, req.Name),
				Namespace:     req.Namespace,
			}); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list eci-records, error: %v", err)
			}
			if len(records.Items) == 0 {
				klog.Infof("update pod return nil %s, records.Items nil", pod.Name)
				return ctrl.Result{}, nil
			}
			slices.SortFunc(records.Items, func(a, b eciv1.PodRecord) int {
				return cmp.Compare(a.Name, b.Name)
			})
			record = records.Items[len(records.Items)-1]
		}
		if record.Name == "" {
			klog.Infof("update pod return nil %s, record name is nil", pod.Name)
			return ctrl.Result{}, nil
		}
		record.Spec.EndTime = time.Now().Format(constants.TimeTemplate)
		record.Spec.EndStatus = status
		if pod.Status.Phase == "" {
			record.Spec.EndStatus = constants.PodStatusDeleted
		}
		if err := r.Update(ctx, &record); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update eci-record, error: %v", err)
		}
		if pod.Labels == nil {
			return ctrl.Result{}, nil
		}
		patch := client.StrategicMergeFrom(pod.DeepCopy())
		delete(pod.Labels, constants.LabelPodEciRecord)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return client.IgnoreNotFound(r.Patch(ctx, &pod, patch))
		}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update pod to add label, error: %v", err)
		}
		//klog.Infof("record %s update success", record.Name)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {

	podMapper := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
		pod, ok := object.(*v1.Pod)
		if !ok {
			return nil
		}

		if r.exclude(r.ExcludeRules, pod) {
			return nil
		}
		//bytes, _ := json.Marshal(pod)
		//klog.Infof("pod %s dont exclude", string(bytes))
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			},
			},
		}
	})
	return ctrl.NewControllerManagedBy(mgr).
		Named(constants.ControllerPodRecord).
		Watches(&v1.Pod{}, podMapper, builder.WithPredicates(&utils.PodStatusChangePredicate{})).
		Complete(r)
}

func (r *PodRecordReconciler) exclude(rules []ExcludeRules, pod *v1.Pod) bool {
	for _, rule := range rules {
		switch rule.Kind {
		case constants.ResourcePod:
			if rule.Name == pod.Name && rule.Namespace == pod.Namespace {
				return true
			}
		case constants.ResourceStatefulSet, constants.ResourceDeployment,
			constants.ResourceDaemonSet, constants.ResourceReplicaSet:
			if r.matchReferences(pod.OwnerReferences, rule, pod.Namespace) {
				return true
			}
		}
	}
	return false
}

func (r *PodRecordReconciler) ownerReferences(reference metav1.OwnerReference, ns string) ([]metav1.OwnerReference, error) {
	if reference.Name == "" {
		return []metav1.OwnerReference{}, nil
	}
	ctx := context.Background()
	var references []metav1.OwnerReference
	nsName := types.NamespacedName{Namespace: ns, Name: reference.Name}
	switch reference.Kind {
	case constants.ResourcePod:
		pod := v1.Pod{}
		if err := r.Get(ctx, nsName, &pod); err != nil {
			return nil, err
		}
		references = pod.OwnerReferences
	case constants.ResourceDeployment:
		deploy := appv1.Deployment{}
		if err := r.Get(ctx, nsName, &deploy); err != nil {
			return nil, err
		}
		references = deploy.OwnerReferences
	case constants.ResourceStatefulSet:
		sts := appv1.StatefulSet{}
		if err := r.Get(ctx, nsName, &sts); err != nil {
			return nil, err
		}
		references = sts.OwnerReferences
	case constants.ResourceReplicaSet:
		rs := appv1.ReplicaSet{}
		if err := r.Get(ctx, nsName, &rs); err != nil {
			return nil, err
		}
		references = rs.OwnerReferences
	case constants.ResourceDaemonSet:
		ds := appv1.DaemonSet{}
		if err := r.Get(ctx, nsName, &ds); err != nil {
			return nil, err
		}
		references = ds.OwnerReferences
	case constants.ResourceJob:
		job := batchv1.Job{}
		if err := r.Get(ctx, nsName, &job); err != nil {
			return nil, err
		}
		references = job.OwnerReferences
	case constants.ResourceCronJob:
		cron := batchv1.CronJob{}
		if err := r.Get(ctx, nsName, &cron); err != nil {
			return nil, err
		}
		references = cron.OwnerReferences
	}

	list := make([]metav1.OwnerReference, 0)
	for i := 0; i < len(references); i++ {
		ref := references[i]
		if ref.Name == "" {
			continue
		}
		if ref.Kind != constants.ResourceDeployment &&
			ref.Kind != constants.ResourceDaemonSet &&
			ref.Kind != constants.ResourceStatefulSet &&
			ref.Kind != constants.ResourceReplicaSet &&
			ref.Kind != constants.ResourcePod {
			continue
		}
		subRef, err := r.ownerReferences(ref, ns)
		if err != nil {
			return nil, err
		}
		list = append(list, subRef...)
	}
	if len(list) == 0 {
		return []metav1.OwnerReference{reference}, nil
	}
	return list, nil
}

func (r *PodRecordReconciler) matchReferences(references []metav1.OwnerReference, rule ExcludeRules, ns string) bool {
	list := make([]metav1.OwnerReference, 0)
	for _, reference := range references {
		if reference.Kind == constants.ResourceNode {
			return true
		}
		refs, err := r.ownerReferences(reference, ns)
		if err != nil {
			klog.Errorf("getReferences error %v:", err)
			return false
		}
		list = append(list, refs...)
	}
	for _, item := range list {
		if rule.Kind == item.Kind && rule.Name == item.Name && ns == rule.Namespace {
			return true
		}
	}
	return false
}

func (r *PodRecordReconciler) newRecord(pod v1.Pod) (*eciv1.PodRecord, error) {

	node := v1.Node{}
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "", Name: pod.Spec.NodeName}, &node); err != nil {
		return nil, fmt.Errorf("failed to get node %s error is %v", pod.Spec.NodeName, err)
	}
	nodeMem := float64(node.Status.Capacity.Memory().Value()) / 1024 / 1024 / 1024
	nodeCpu := node.Status.Capacity.Cpu()
	cpuLimit := float64(utils.PodCpuLimit(pod)) / 1000
	if cpuLimit == 0 {
		cpuLimit = float64(nodeCpu.Value())
	}
	memLimit := float64(utils.PodMemLimit(pod)) / 1024 / 1024 / 1024
	if memLimit == 0 {
		memLimit = nodeMem
	}
	cpuRequest := float64(utils.PodCpuRequest(pod)) / 1000
	memRequest := float64(utils.PodMemRequest(pod)) / 1024 / 1024 / 1024
	//memRequest = float64(memRequest)
	return &eciv1.PodRecord{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%d", time.Now().UnixNano()),
			Namespace: pod.Namespace,
		},

		Spec: eciv1.PodRecordSpec{
			UserID:  r.CustomerID,
			PodID:   string(pod.UID),
			PodName: pod.Name,
			//CpuRequest: strconv.FormatInt(cpuRequest, 10),
			CpuRequest: fmt.Sprintf("%.2f", utils.Round(cpuRequest, 2)),
			MemRequest: fmt.Sprintf("%.2f", utils.Round(memRequest, 2)),
			//CpuLimit:   strconv.FormatInt(cpuLimit, 10),
			CpuLimit: fmt.Sprintf("%.2f", utils.Round(cpuLimit, 2)),
			MemLimit: fmt.Sprintf("%.2f", utils.Round(memLimit, 2)),
			Gpu:      utils.PodGpuNvidia(pod),
			Node:     node.Name,
			NodeMem:  fmt.Sprintf("%.2f", utils.Round(nodeMem, 2)),
			NodeCpu:  nodeCpu.String(),
			//StartTime:  pod.Status.StartTime.Time.Format(constants.TimeTemplate),
			StartTime: time.Now().Format(constants.TimeTemplate),
		},
	}, nil
}

func containersRunning(pod v1.Pod) bool {
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running == nil {
			return false
		}
	}
	return true
}

func podStatus(pod v1.Pod) string {
	if !containersRunning(pod) && pod.Status.Phase == constants.PodStatusRunning {
		return constants.StatusContainerNotRunning
	}
	return string(pod.Status.Phase)
}

type ExcludeRules struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Condition string `json:"condition"`
}
