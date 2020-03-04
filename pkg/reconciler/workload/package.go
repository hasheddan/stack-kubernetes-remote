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

package workload

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane/apis/workload/v1alpha1"
)

// A Packager is responsible for packaging objects into a KubernetesApplication.
type Packager interface {
	Package(context.Context, *v1alpha1.KubernetesApplication, runtime.Object) error
}

// A PackageFn packages an object into a KubernetesApplication.
type PackageFn func(context.Context, *v1alpha1.KubernetesApplication, runtime.Object) error

// Package object into KubernetesApplication.
func (fn PackageFn) Package(ctx context.Context, app *v1alpha1.KubernetesApplication, obj runtime.Object) error {
	return fn(ctx, app, obj)
}

func genericPackager(ctx context.Context, app *v1alpha1.KubernetesApplication, obj runtime.Object) error {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	kart := &v1alpha1.KubernetesApplicationResourceTemplate{
		Spec: v1alpha1.KubernetesApplicationResourceSpec{
			Template: &unstructured.Unstructured{Object: u},
		},
	}

	app.Spec.ResourceTemplates = append(app.Spec.ResourceTemplates, *kart)
	return nil
}
