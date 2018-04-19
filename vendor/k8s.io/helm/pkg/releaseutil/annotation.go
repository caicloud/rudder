/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package releaseutil

import (
	"bytes"
	"log"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v1beta1"
	app "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

// AnnotationKey is annotation key of kubernetes object
type AnnotationKey string

const (
	// DefaultPathKey is the key of chart logic path. Logic path splits components by
	// slash. For example:
	// A chart like:
	// rootchart --> subchart1
	//           |-> subchart2
	// The logic path of every charts:
	// rootchart: rootchart
	// subchart1: rootchart/subchart1
	// subchart2: rootchart/subchart2
	// If a kubernetes resource object have an annotation key named "helm.sh/path", Then
	// its value should meet the requirements.
	DefaultPathKey AnnotationKey = "helm.sh/path"
	// defaultNamespaceKey is the key of release namespace
	DefaultNamespaceKey AnnotationKey = "helm.sh/namespace"
	// defaultReleaseKey is the key of release name
	DefaultReleaseKey AnnotationKey = "helm.sh/release"
)

var (
	// defaultSerializer is a codec and used for encoding and decoding kubernetes resources
	defaultSerializer *json.Serializer = nil
)

func init() {
	// create default serializer
	defaultSerializer = json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
}

// InjectAnnotations adds key-value pairs to resource annotations. If resource is not a valid kubernetes
// resource, it does nothing and returns original resource.
func InjectAnnotations(resource string, annos map[AnnotationKey]string) string {
	if len(annos) <= 0 {
		return resource
	}

	// decode object
	obj, _, err := defaultSerializer.Decode([]byte(resource), nil, nil)
	if err != nil {
		return resource
	}
	accessor := meta.NewAccessor()
	annotations, err := accessor.Annotations(obj)
	if err != nil {
		return resource
	}
	err = accessor.SetAnnotations(obj, merge(annotations, annos))
	if err != nil {
		return resource
	}

	// check and add annotations to the template of specific types
	switch ins := obj.(type) {
	case *apps.Deployment:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	case *apps.DaemonSet:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	case *apps.ReplicaSet:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	case *apps.StatefulSet:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	case *batch.Job:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	case *batchv2.CronJob:
		{
			ins.Spec.JobTemplate.Annotations = merge(ins.Spec.JobTemplate.Annotations, annos)
			ins.Spec.JobTemplate.Spec.Template.Annotations = merge(ins.Spec.JobTemplate.Spec.Template.Annotations, annos)
		}
	case *app.ReplicationController:
		{
			ins.Spec.Template.Annotations = merge(ins.Spec.Template.Annotations, annos)
		}
	}

	// encode object
	buf := bytes.NewBuffer(nil)
	err = defaultSerializer.Encode(obj, buf)
	if err != nil {
		return resource
	}
	return buf.String()
}

// merge merges annotations into origin
func merge(origin map[string]string, annos map[AnnotationKey]string) map[string]string {
	if origin == nil {
		origin = make(map[string]string)
	}
	for k, v := range annos {
		origin[string(k)] = v
	}
	return origin
}

// MatchReleaseByString matches two object string and check if they are from same release
func MatchReleaseByString(a, b string) bool {
	// decode object
	objA, _, err := defaultSerializer.Decode([]byte(a), nil, nil)
	if err != nil {
		log.Printf("match: Failed to decode resource: %s", err)
		return false
	}
	objB, _, err := defaultSerializer.Decode([]byte(b), nil, nil)
	if err != nil {
		log.Printf("match: Failed to decode resource: %s", err)
		return false
	}
	return MatchRelease(objA, objB)
}

// MatchRelease matches two object and check if they are from same release
func MatchRelease(a, b runtime.Object) bool {
	accessor := meta.NewAccessor()
	annoA, err := accessor.Annotations(a)
	if err != nil {
		annoA = map[string]string{}
	}
	annoB, err := accessor.Annotations(b)
	if err != nil {
		annoB = map[string]string{}
	}
	return matchReleaseAnnotations(annoA, annoB)
}

// matchReleaseAnnotations checks path, namespace and release
func matchReleaseAnnotations(a, b map[string]string) bool {
	keys := []string{string(DefaultPathKey), string(DefaultNamespaceKey), string(DefaultReleaseKey)}
	for _, key := range keys {
		v1, ok1 := a[key]
		v2, ok2 := b[key]
		if !(ok1 || ok2) {
			continue
		}
		if v1 != v2 {
			return false
		}
	}
	return true
}
