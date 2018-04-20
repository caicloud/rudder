# clientset
A set of kubernetes api client for all native resources and tprs

Usage:

1. Defines types in `pkg/apis/` (Add client/informer/lister expansions to `expansions`)
2. Run command: `make` to generate client/informer/lister


For every versioned API:
1. `doc.go`: package-level tags
  - `// +k8s:deepcopy-gen=package`
    - [Required] Generating deepcopy methods for types.
  - `// +k8s:defaulter-gen=TypeMeta`
    - [Optional] Generating default methods for types (`func SetDefaults_Release(obj *Release)`).
  - `// +k8s:conversion-gen=github.com/caicloud/clientset/pkg/apis/release`
    - [Optional] Generating convension methods for types (`func Convert_v1alpha1_Release_To_release_Release(in *Release, out *release.Release, s conversion.Scope) error`).
  - `// +groupName=release.caicloud.io`
    - [Required] Used in the fake client as the full group name.
2. `types.go`: type-level tags
  - `// +genclient`
    - [Required] Generateing clients for types
  - `// +genclient:noStatus`
    - [Required] Generateing clients for types without method `UpdateStatus`.
  - `// +genclient:onlyVerbs=create,get`
    - [Optional] Generateing clients with specific verbs.
  - `// +genclient:skipVerbs=watch`
    - [Optional] Generateing clients without specific verbs.
  - `// +genclient:nonNamespaced`
    - [Optional] Generating global types rather than namespaced types.
  - `// +genclient:method=Scale,verb=update,subresource=scale,input=k8s.io/api/extensions/v1beta1.Scale,result=k8s.io/api/extensions/v1beta1.Scale`
    - [Optional] Generating external method.
  - `// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`
    - [Required] Generating deepcopy methods with implementing `runtime.Object`.
