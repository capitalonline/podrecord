/*
Copyright 2023.

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

package utils

import (
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	// RuntimeLog is a base parent logger for use inside controller-runtime.
	RuntimeLog logr.Logger
)

func init() {
	RuntimeLog = log.Log.WithName("controller-runtime")
}

var logger = RuntimeLog.WithName("predicate").WithName("eventFilters")

type PodStatusChangePredicate struct {
	predicate.Funcs
}

func (PodStatusChangePredicate) Update(e event.UpdateEvent) bool {
	oldPod, ok := e.ObjectOld.(*v1.Pod)
	if !ok {
		logger.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}

	newPod, ok := e.ObjectNew.(*v1.Pod)
	if !ok {
		logger.Error(nil, "Update event has no new object for update", "event", e)
		return false
	}

	return !reflect.DeepEqual(oldPod.Status.Phase, newPod.Status.Phase)
}
