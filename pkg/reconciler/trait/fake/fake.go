/*
Copyright 2020 The Crossplane Authors.

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

package fake

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
)

// Trait is a mock that implements Trait interface.
type Trait struct {
	metav1.ObjectMeta
	v1alpha1.ConditionedStatus
}

// GetWorkloadRef gets the workload ref.
func (t *Trait) GetWorkloadRef() corev1.ObjectReference {
	return corev1.ObjectReference{}
}

// GetObjectKind returns schema.ObjectKind.
func (t *Trait) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

// DeepCopyObject returns a copy of the object as runtime.Object
func (t *Trait) DeepCopyObject() runtime.Object {
	out := &Trait{}
	j, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(j, out)
	return out
}
