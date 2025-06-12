package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ParcaScrapeConfigSpec defines the desired state of ParcaScrapeConfig
type ParcaScrapeConfigSpec struct {
	// Selector is the selector for the Pods which should be scraped by Parca.
	Selector metav1.LabelSelector `json:"selector"`
	// ScrapeConfig is the scrape configuration as it can be set in the Parca
	// configuration.
	ScrapeConfig ScrapeConfig `json:"scrapeConfig,omitempty"`
}

type ScrapeConfig struct {
	// Job is the job name of the section in the configurtion. If no job name
	// is provided, it will be automatically generated based on the name and
	// namespace of the CR: "namespace/name"
	Job string `json:"job,omitempty"`
	// Port is the name of the port of the Pods which is used to expose the
	// profiling endpoints.
	Port string `json:"port,omitempty"`
	// PortNumber is the number of the port which is used to expose the
	// profiling endpoints. This can be used instead of the port field. If the
	// port is not named.
	PortNumber int64 `json:"portNumber,omitempty"`
	// Params is a set of query parameters with which the target is scraped.
	Params map[string][]string `json:"params,omitempty"`
	// Interval defines how frequently to scrape the targets of this scrape
	// config.
	Interval Duration `json:"interval,omitempty"`
	// Timeout defines the timeout for scraping targets of this config.
	Timeout Duration `json:"timeout,omitempty"`
	// Schema sets the URL scheme with which to fetch metrics from targets.
	Scheme string `json:"scheme,omitempty"`
	// ProfilingConfig defines the profiling config for the targets, see
	// https://www.parca.dev/docs/ingestion#pull-based for more information.
	ProfilingConfig *ProfilingConfig `json:"profilingConfig,omitempty"`
	// RelabelConfigs allows dynamic rewriting of the label set for the targets,
	// see https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config
	// for more information.
	RelabelConfigs []RelabelConfig `json:"relabelConfigs,omitempty"`
}

// Duration is a valid time duration that can be parsed by Prometheus
// model.ParseDuration() function.
//
// Supported units: y, w, d, h, m, s, ms
// Examples: `30s`, `1m`, `1h20m15s`, `15d`
//
// +kubebuilder:validation:Pattern:="^(0|(([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?)$"
type Duration string

type ProfilingConfig struct {
	PprofConfig PprofConfig `json:"pprofConfig,omitempty"`
	PprofPrefix string      `json:"pathPrefix,omitempty"`
}

type PprofConfig map[string]*PprofProfilingConfig

type PprofProfilingConfig struct {
	Enabled        *bool        `json:"enabled,omitempty"`
	Path           string       `json:"path,omitempty"`
	Delta          bool         `json:"delta,omitempty"`
	KeepSampleType []SampleType `json:"keepSampleType,omitempty"`
	Seconds        int          `json:"seconds,omitempty"`
}

type SampleType struct {
	Type string `json:"type,omitempty"`
	Unit string `json:"unit,omitempty"`
}

// ParcaScrapeConfigStatus defines the observed state of ParcaScrapeConfig
type ParcaScrapeConfigStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// LabelName is a valid Prometheus label name which may only contain ASCII
// letters, numbers, as well as underscores.
//
// +kubebuilder:validation:Pattern:="^[a-zA-Z_][a-zA-Z0-9_]*$"
type LabelName string

// RelabelConfig allows dynamic rewriting of the label set for targets, alerts,
// scraped samples and remote write samples.
//
// More info: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config
//
// +k8s:openapi-gen=true
type RelabelConfig struct {
	// The source labels select values from existing labels. Their content is
	// concatenated using the configured Separator and matched against the
	// configured regular expression.
	//
	// +optional
	SourceLabels []LabelName `json:"sourceLabels,omitempty"`
	// Separator is the string between concatenated SourceLabels.
	Separator string `json:"separator,omitempty"`
	// Label to which the resulting string is written in a replacement.
	//
	// It is mandatory for `Replace`, `HashMod`, `Lowercase`, `Uppercase`,
	// `KeepEqual` and `DropEqual` actions.
	//
	// Regex capture groups are available.
	TargetLabel string `json:"targetLabel,omitempty"`
	// Regular expression against which the extracted value is matched.
	Regex string `json:"regex,omitempty"`
	// Modulus to take of the hash of the source label values.
	//
	// Only applicable when the action is `HashMod`.
	Modulus uint64 `json:"modulus,omitempty"`
	// Replacement value against which a Replace action is performed if the
	// regular expression matches.
	//
	// Regex capture groups are available.
	//
	//+optional
	Replacement string `json:"replacement,omitempty"`
	// Action to perform based on the regex matching.
	//
	// `Uppercase` and `Lowercase` actions require Prometheus >= v2.36.0.
	// `DropEqual` and `KeepEqual` actions require Prometheus >= v2.41.0.
	//
	// Default: "Replace"
	//
	// +kubebuilder:validation:Enum=replace;Replace;keep;Keep;drop;Drop;hashmod;HashMod;labelmap;LabelMap;labeldrop;LabelDrop;labelkeep;LabelKeep;lowercase;Lowercase;uppercase;Uppercase;keepequal;KeepEqual;dropequal;DropEqual
	// +kubebuilder:default=replace
	Action string `json:"action,omitempty"`
}

//+kubebuilder:printcolumn:name="Succeeded",type=string,JSONPath=`.status.conditions[?(@.type=="ParcaScrapeConfigReconciled")].status`,description="Indicates if the Parca scrape configuration was updated successfully"
//+kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="ParcaScrapeConfigReconciled")].reason`,description="Reason for the current status"
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="ParcaScrapeConfigReconciled")].message`,description="Message with more information, regarding the current status"
//+kubebuilder:printcolumn:name="Last Transition",type=date,JSONPath=`.status.conditions[?(@.type=="ParcaScrapeConfigReconciled")].lastTransitionTime`,description="Time when the condition was updated the last time"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time when this ParcaScrapeConfigration was updated the last time"
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ParcaScrapeConfig is the Schema for the parcascrapeconfigs API
type ParcaScrapeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ParcaScrapeConfigSpec   `json:"spec,omitempty"`
	Status ParcaScrapeConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ParcaScrapeConfigList contains a list of ParcaScrapeConfig
type ParcaScrapeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ParcaScrapeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ParcaScrapeConfig{}, &ParcaScrapeConfigList{})
}
