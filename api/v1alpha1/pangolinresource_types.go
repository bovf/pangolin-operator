package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SiteReference allows referencing a Pangolin site either by numeric ID or nice ID
// Priority: SiteID takes precedence over NiceID if both are specified
type SiteReference struct {
	// SiteID is the numeric site identifier
	// +optional
	SiteID *int `json:"siteId,omitempty"`

	// NiceID is the human-readable site identifier
	// +optional
	NiceID string `json:"niceId,omitempty"`
}

// PangolinResourceSpec defines the desired state of PangolinResource
type PangolinResourceSpec struct {
	// Reference to the tunnel this resource belongs to
	// +optional
	TunnelRef LocalObjectReference `json:"tunnelRef,omitempty"`

	// SiteRef directly references a Pangolin site
	// If specified, takes precedence over TunnelRef for site resolution
	// +optional
	SiteRef *SiteReference `json:"siteRef,omitempty"`

	// Resource configuration for NEW resources
	Name     string `json:"name,omitempty"`
	Protocol string `json:"protocol,omitempty"`

	// BINDING MODE: Resource ID to bind to existing resource
	// If specified, will bind to existing resource instead of creating new one
	ResourceID string `json:"resourceId,omitempty"`

	// HTTP-specific configuration
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`

	// TCP/UDP-specific configuration
	ProxyConfig *ProxyConfig `json:"proxyConfig,omitempty"`

	// Targets configuration - multiple targets for path-based routing
	// +optional
	Targets []TargetConfig `json:"targets,omitempty"`

	// Enable/disable this resource
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// HTTPConfig defines HTTP-specific resource configuration - ENHANCED
type HTTPConfig struct {
	// Subdomain for this resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Subdomain string `json:"subdomain"`

	// OPTION 1: Domain ID to use (e.g., "domain1")
	DomainID string `json:"domainId,omitempty"`

	// OPTION 2: Domain name to use (e.g., "yourdomain.com") - NEW
	// If specified, will be resolved to domainId by the operator
	DomainName string `json:"domainName,omitempty"`

	// If neither domainId nor domainName is specified,
	// will use the organization's default domain

	// SSO enables SSO authentication for this resource
	// +optional
	SSO bool `json:"sso"`

	// BlockAccess blocks access until user is authenticated
	// Only effective when SSO is enabled
	// +optional
	BlockAccess bool `json:"blockAccess"`
}

// ProxyConfig defines TCP/UDP proxy configuration
type ProxyConfig struct {
	// Proxy port to expose
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ProxyPort int32 `json:"proxyPort"`
	// Enable proxy functionality
	// +kubebuilder:default=true
	EnableProxy *bool `json:"enableProxy,omitempty"`
}

// TargetConfig defines the backend target
type TargetConfig struct {
	// Target IP or hostname
	// +kubebuilder:validation:Required
	IP string `json:"ip"`
	// Target port
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
	// Target method/protocol
	// +kubebuilder:validation:Enum=http;https;tcp;udp
	// +kubebuilder:default="http"
	Method string `json:"method,omitempty"`
	// Path to match for routing (e.g., "/api")
	// +optional
	Path string `json:"path,omitempty"`
	// PathMatchType defines how to match the path: exact, prefix, or regex
	// +kubebuilder:validation:Enum=exact;prefix;regex
	// +optional
	PathMatchType string `json:"pathMatchType,omitempty"`
	// Priority for path matching (higher = matched first)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// +kubebuilder:default=100
	// +optional
	Priority int32 `json:"priority,omitempty"`
}

// LocalObjectReference contains enough information to locate a resource
type LocalObjectReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the referenced resource. If empty, defaults to the same namespace as the referencing resource.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// PangolinResourceStatus defines the observed state of PangolinResource
// PangolinResourceStatus defines the observed state of PangolinResource
type PangolinResourceStatus struct {
	// Resource ID from Pangolin API
	ResourceID string `json:"resourceId,omitempty"`

	// DEPRECATED: Keep for backward compatibility but don't use
	TargetID string `json:"targetId,omitempty"`

	// All target IDs for this resource
	TargetIDs []string `json:"targetIds,omitempty"`

	// Resolved domain ID from domain name
	ResolvedDomainID string `json:"resolvedDomainId,omitempty"`

	// Full domain where resource is accessible
	FullDomain string `json:"fullDomain,omitempty"`

	// Binding mode: "Created" or "Bound"
	BindingMode string `json:"bindingMode,omitempty"`

	// Current status: Creating, Ready, Error, Deleting, Waiting
	// +kubebuilder:validation:Enum=Creating;Ready;Error;Deleting;Waiting
	Status string `json:"status,omitempty"`

	// Public URL for HTTP resources
	URL string `json:"url,omitempty"`

	// Proxy endpoint for TCP/UDP resources
	ProxyEndpoint string `json:"proxyEndpoint,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// SSOEnabled indicates if SSO authentication is enabled for this resource
	SSOEnabled bool `json:"ssoEnabled,omitempty"`

	// BlockAccessEnabled indicates if access is blocked until authenticated
	BlockAccessEnabled bool `json:"blockAccessEnabled,omitempty"`

	// TargetCount is the number of targets configured for this resource
	TargetCount int `json:"targetCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=presource
//+kubebuilder:printcolumn:name="Resource ID",type=string,JSONPath=`.status.resourceId`
//+kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
//+kubebuilder:printcolumn:name="Subdomain",type=string,JSONPath=`.spec.httpConfig.subdomain`
//+kubebuilder:printcolumn:name="Full Domain",type=string,JSONPath=`.status.fullDomain`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="Binding Mode",type=string,JSONPath=`.status.bindingMode`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PangolinResource is the Schema for the pangolinresources API
type PangolinResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PangolinResourceSpec   `json:"spec,omitempty"`
	Status PangolinResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PangolinResourceList contains a list of PangolinResource
type PangolinResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PangolinResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PangolinResource{}, &PangolinResourceList{})
}
