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

package trait

import (
	"context"

	"github.com/crossplane/crossplane/apis/workload/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// A Modifier is responsible for modifying or adding objects in a KubernetesApplication.
type Modifier interface {
	Modify(context.Context, *v1alpha1.KubernetesApplication, runtime.Object) error
}

// A ModifyFn modifies or adds an object in a KubernetesApplication.
type ModifyFn func(context.Context, *v1alpha1.KubernetesApplication, runtime.Object) error

// Modify object in KubernetesApplication.
func (fn ModifyFn) Modify(ctx context.Context, app *v1alpha1.KubernetesApplication, obj runtime.Object) error {
	return fn(ctx, app, obj)
}

func noopModifier(_ context.Context, _ *v1alpha1.KubernetesApplication, _ runtime.Object) error {
	return nil
}
