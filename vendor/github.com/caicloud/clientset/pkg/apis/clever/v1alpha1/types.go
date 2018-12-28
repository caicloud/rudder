package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicaType string
type StagePhase string
type StageID string
type DatasetType string
type ToolType string
type FrameworkType string
type ProtocolType string
type ProjectPhase string
type TemplateType string

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
	ServingType FrameworkType = "serving"
)

const (
	FlavorPlural   = "flavors"
	ProjectPlural  = "projects"
	TemplatePlural = "templates"
)

const (
	StageReady       StagePhase = "Ready"
	StageCreating    StagePhase = "Creating"
	StageError       StagePhase = "Error"
	StageTerminating StagePhase = "Terminating"
)

const (
	Model DatasetType = "model"
	Data  DatasetType = "dataset"
)

const (
	JupyterLab      ToolType = "jupyterLab"
	JupyterNotebook ToolType = "jupyterNotebook"
)

const (
	ProjectReady       ProjectPhase = "Ready"
	ProjectFailed      ProjectPhase = "Failed"
	ProjectTerminating ProjectPhase = "Terminating"
)

const (
	TrainingTemplate TemplateType = "training"
	GeneralTemplate  TemplateType = "general"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is a custom resource definition CRD contains all steps and stages
// which can do training model, serving model and some custom job.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status"`
}

// ProjectSpec defines the specification for a project.
type ProjectSpec struct {
	// Stages defines all offline stages in a project.
	Stages []Stage `json:"stages"`
	// Steps defines all offline steps in a project.
	Steps []Step `json:"steps"`
	// Tools contains all the tools used in a project, e.g. jupyter, tensorboard, etc
	Tools []Tool `json:"tools"`
	// Storage contains all storage used in a project.
	Storage []ProjectStorage `json:"storage"`
}

type ProjectStorage struct {
	VolumeSource corev1.VolumeSource `json:"volumeSource"`
	Size         string              `json:"size"`
}

type ProjectStatus struct {
	Phase      ProjectPhase           `json:"phase"`
	StagePhase map[StageID]StagePhase `json:"stagePhase"`
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
	Name string `json:"name"`
	// Tool's type, include jupyter, jupyter lab
	Type ToolType `json:"type"`
	// Tool's image
	Image ImageFlavor `json:"image"`
	// Tool's resource
	Resource ResourceFlavor `json:"resource"`
	// Tool's Env
	Env []corev1.EnvVar `json:"env"`
}

// Stage defines a single offline stage in a project: it is a template
// specification with values filled in.
type Stage struct {
	// StageMeta contains metadata of an offline stage.
	StageMeta `json:",inline"`
	// StepUID references the step that this stage belongs to.
	StepUID string `json:"stepUID"`
	// Template with configuration filled in.
	Template TemplateSpec `json:"template"`
	// Tool references to the tools available in this template.
	ToolID []string `json:"toolID"`
}

type StageMeta struct {
	Username     string            `json:"userName"`
	UID          string            `json:"uid"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	CreationTime metav1.Time       `json:"creationTime"`
	Annotations  map[string]string `json:"annotations"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Project `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Template CRD
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TemplateSpec `json:"spec"`
}

// TemplateSpec contains all necessary information to define a template.
type TemplateSpec struct {
	// TemplateSource the source of the template, which is categorized into different types.
	TemplateSource `json:",inline"`
	// Flavor references to the flavors available this template.
	Flavors []string `json:"flavors"`
	// Properties is a Template property contains logo, type and framework.
	Properties Properties `json:"properties"`
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
	Logo string `json:"logo"`
	// Type defines the type of the template. e.g. training, serving, etc.
	Type TemplateType `json:"type"`
	// Framework defines the framework of the template. e.g. tensorflow, pytorch, etc.
	Framework FrameworkType `json:"framework"`
}

type Training struct {
	// Inputs dataset for a training stage.
	Inputs []Dataset `json:"inputs"`
	// Outputs dataset for a training stage.
	Outputs []Dataset `json:"outputs"`
	// Image used in the training stage.
	Image ImageFlavor `json:"image"`
	// Replicas used in training stage.
	Replicas []Replica `json:"replicas"`
	// Pod's command
	Command string `json:"command"`
	// Pod's workdir
	WorkDir string `json:"workdir"`
	// Pod's codedir
	CodeDir string `json:"codedir"`
	// Pod's env
	Env []corev1.EnvVar `json:"env"`
	// Dependence files
	Dependency Dependency `json:"dependency,omitempty"`
}

type Serving struct {
	// Model is a model's info in model set.
	Model ModelInfo `json:"model"`
	// Developments is development environment serving list.
	Developments []string `json:"developments"`
	// Productions is production environment serving list.
	Productions []string `json:"productions"`
}

type General struct {
	// Inputs dataset for a general stage.
	Inputs []Dataset `json:"inputs"`
	// Outputs dataset for a general stage.
	Outputs []Dataset `json:"outputs"`
	// Image used in general stage.
	Image ImageFlavor `json:"image"`
	// Replica used in general stage.
	Replica Replica `json:"replica"`
	// Pod's command
	Command string `json:"command"`
	// Pod's workdir
	WorkDir string `json:"workdir"`
	// Pod's codedir
	CodeDir string `json:"codedir"`
	// Pod's env
	Env []corev1.EnvVar `json:"env"`
	// Dependence files
	Dependency Dependency `json:"dependency"`
}

type ModelInfo struct {
	Name      string        `json:"name"`
	Framework FrameworkType `json:"framework"`
}

// DataSet is struct of Projects Input and Output
type Dataset struct {
	Name    string      `json:"name"`
	Type    DatasetType `json:"type"`
	Version string      `json:"version"`
}

type Dependency struct {
	Path  string `json:"path,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Flavors CRD
type Flavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FlavorSpec `json:"spec"`
}

type FlavorSpec struct {
	// Template images, can be selected
	Images []ImageFlavor `json:"images"`
	// Template resource, can be selected
	Resources []ResourceFlavor `json:"resources"`
}

type ImageFlavor struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Builtin bool   `json:"builtin"`
}

type Replica struct {
	Type     ReplicaType    `json:"type"`
	Count    int32          `json:"count"`
	Resource ResourceFlavor `json:"resource"`
}

type ResourceFlavor struct {
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    string `json:"gpu"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flavor `json:"items"`
}
