package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicaType string
type StageStatus string
type StageID string
type DatasetType string
type ToolType string
type FrameworkType string
type ProtocolType string

const (
	ProtocolgRPC    ProtocolType = "gRPC"
	ProtocolRESTful ProtocolType = "RESTful"
)

const (
	ReplicaTypeMaster ReplicaType = "master"
	ReplicaTypeWorker ReplicaType = "worker"
	ReplicaTypeEval   ReplicaType = "eval"
	ReplicaTypePS     ReplicaType = "ps"
	ReplicaTypeChief  ReplicaType = "chief"
)

const (
	Python      FrameworkType = "python"
	Clang       FrameworkType = "clang"
	Chainer     FrameworkType = "chainer"
	CPP         FrameworkType = "cpp"
	Golang      FrameworkType = "golang"
	Java        FrameworkType = "java"
	Tensorflow  FrameworkType = "tensorflow"
	Pytorch     FrameworkType = "pytorch"
	Caffe       FrameworkType = "caffe"
	Caffe2      FrameworkType = "caffe2"
	MXNet       FrameworkType = "mxnet"
	Keras       FrameworkType = "keras"
	SKLearn     FrameworkType = "sklearn"
	TFserving   FrameworkType = "tfserving"
	OnnxServing FrameworkType = "onnxserving"
)

const (
	FlavorPlural   = "flavors"
	ProjectPlural  = "projects"
	TemplatePlural = "templates"
)

const (
	StageReady    StageStatus = "Ready"
	StageCreating StageStatus = "Creating"
	StageError    StageStatus = "Error"
)

const (
	Model DatasetType = "model"
	Data  DatasetType = "data"
)

const (
	JupyterLab      ToolType = "jupyterLab"
	JupyterNotebook ToolType = "jupyterNotebook"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is a custom resource definition CRD contains all steps and stages
// which can do training model, serving model and some custom job.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec defines the specification for a project.
type ProjectSpec struct {
	// Stages defines all offline stages in a project.
	Stages []Stage `json:"stages,omitempty"`
	// Steps defines all offline steps in a project.
	Steps []Step `json:"steps,omitempty"`
	// Tools contains all the tools used in a project, e.g. jupyter, tensorboard, etc
	Tools []Tool `json:"tools,omitempty"`
	// Storage contains all storage used in a project.
	Storage []corev1.VolumeSource `json:"storage,omitempty"`
}

type ProjectStatus struct {
	StageStatus map[StageID]StageStatus `json:"stageStatus,omitempty"`
}

type Step struct {
	UID          string      `json:"uid,omitempty"`
	Name         string      `json:"name"`
	CreationTime metav1.Time `json:"creationTime"`
}

type Tool struct {
	// Tool's uid
	UID string `json:"uid"`
	// Tool's name
	Name string `json:"name,omitempty"`
	// Tool's type, include jupyter, jupyter lab
	Type ToolType `json:"type,omitempty"`
	// Tool's image
	Image ImageFlavor `json:"image,omitempty"`
	// Tool's resource
	Resource ResourceFlavor `json:"resource,omitempty"`
	// Tool's Env
	Env []corev1.EnvVar `json:"env"`
}

// Stage defines a single offline stage in a project: it is a template
// specification with values filled in.
type Stage struct {
	// StageMeta contains metadata of an offline stage.
	StageMeta `json:",inline"`
	// StepUID references the step that this stage belongs to.
	StepUID string `json:"stepUID,omitempty"`
	// Template with configuration filled in.
	Template TemplateSpec `json:"template,omitempty"`
	// Tool references to the tools available in this template.
	ToolID []string `json:"toolID,omitempty"`
}

type StageMeta struct {
	Username     string      `json:"userName,omitempty"`
	UID          string      `json:"uid,omitempty"`
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	CreationTime metav1.Time `json:"creationTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Template CRD
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TemplateSpec `json:"spec,omitempty"`
}

// TemplateSpec contains all necessary information to define a template.
type TemplateSpec struct {
	// TemplateSource the source of the template, which is categorized into different types.
	TemplateSource `json:",inline"`
	// Flavor references to the flavors available this template.
	Flavors []string `json:"flavors,omitempty"`
	// Properties is a Template property contains logo, type and framework.
	Properties Properties `json:"properties,omitempty"`
}

type TemplateSource struct {
	// Training defines a training specification.
	Training *Training `json:"training,omitempty"`
	// Serving defines a serving specification.
	Serving *Serving `json:"serving,omitempty"`
	// General defines a general task specification.
	General *General `json:"general,omitempty"`
}

type Properties struct {
	// Logo defines the logo of the template.
	Logo string `json:"logo,omitempty"`
	// Type defines the type of the template. e.g. training, serving, etc.
	Type string `json:"type,omitempty"`
	// Framework defines the framework of the template. e.g. tensorflow, pytorch, etc.
	Framework FrameworkType `json:"framework,omitempty"`
}

type Training struct {
	// Inputs dataset for a training stage.
	Inputs []Dataset `json:"inputs,omitempty"`
	// Outputs dataset for a training stage.
	Outputs []Dataset `json:"outputs,omitempty"`
	// Image used in the training stage.
	Image ImageFlavor `json:"image,omitempty"`
	// Replicas used in training stage.
	Replicas []Replica `json:"replicas"`
	// Pod's command
	Command string `json:"command,omitempty"`
	// Pod's workdir
	WorkDir string `json:"workdir,omitempty"`
	// Pod's codedir
	CodeDir string `json:"codedir,omitempty"`
	// Pod's env
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Dependence files
	Dependency Dependency `json:"dependency,omitempty"`
}

type Serving struct {
	// Inputs dataset for a serving stage.
	Inputs []Dataset `json:"inputs,omitempty"`
	// Image used in serving stage
	Image ImageFlavor `json:"image,omitempty"`
	// Replica used in serving stage.
	Replica Replica `json:"replica"`
	// Protocol is protocol used in serving model. e.g. gRPC, RESTful.
	Protocol []ProtocolType `json:"protocol"`
}

type General struct {
	// Inputs dataset for a general stage.
	Inputs []Dataset `json:"inputs,omitempty"`
	// Outputs dataset for a general stage.
	Outputs []Dataset `json:"outputs,omitempty"`
	// Image used in general stage.
	Image ImageFlavor `json:"image,omitempty"`
	// Replica used in general stage.
	Replica Replica `json:"replica"`
	// Pod's command
	Command string `json:"command,omitempty"`
	// Pod's workdir
	WorkDir string `json:"workdir,omitempty"`
	// Pod's codedir
	CodeDir string `json:"codedir,omitempty"`
	// Pod's env
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Dependence files
	Dependency Dependency `json:"dependency,omitempty"`
}

// DataSet is struct of Projects Input and Output
type Dataset struct {
	Name    string      `json:"name,omitempty"`
	Type    DatasetType `json:"type,omitempty"`
	Version string      `json:"version,omitempty"`
}

type Dependency struct {
	Path  string `json:"path,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Flavors CRD
type Flavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FlavorSpec `json:"spec,omitempty"`
}

type FlavorSpec struct {
	// Template images, can be selected
	Images []ImageFlavor `json:"images,omitempty"`

	// Template resource, can be selected
	Resources []ResourceFlavor `json:"resources,omitempty"`
}

type ImageFlavor struct {
	Name  string `json:"name,omitempty"`
	Image string `json:"image,omitempty"`
}

type Replica struct {
	Type     ReplicaType    `json:"type,omitempty"`
	Count    int32          `json:"count,omitempty"`
	Resource ResourceFlavor `json:"resource,omitempty"`
}

type ResourceFlavor struct {
	Name   string `json:"name,omitempty"`
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	GPU    string `json:"gpu,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flavor `json:"items,omitempty"`
}
