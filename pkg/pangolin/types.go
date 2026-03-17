package pangolin

import (
	"strconv"
)

// Organization represents a Pangolin organization
type Organization struct {
	OrgID  string `json:"orgId"`
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
}

// Domain represents a Pangolin domain
type Domain struct {
	DomainID      string `json:"domainId"`
	BaseDomain    string `json:"baseDomain"`
	Verified      bool   `json:"verified"`
	Type          string `json:"type"`
	Failed        bool   `json:"failed"`
	Tries         int    `json:"tries"`
	ConfigManaged bool   `json:"configManaged"`
}

// Site represents a Pangolin site (matches Integration API fields)
type Site struct {
	// IDs from API
	SiteID int    `json:"siteId"`
	NiceID string `json:"niceId"`
	OrgID  string `json:"orgId"`

	// Basic info
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status,omitempty"`

	// Network info
	Subnet  string `json:"subnet"`
	Address string `json:"address"`

	// Connection info
	Online   bool   `json:"online"`
	Endpoint string `json:"endpoint"`

	// Keys and security
	PubKey    string `json:"pubKey"`
	PublicKey string `json:"publicKey"`

	// Newt-specific fields
	NewtID              string `json:"newtId,omitempty"`
	NewtSecretKey       string `json:"newtSecretKey,omitempty"`
	NewtVersion         string `json:"newtVersion"`
	NewtUpdateAvailable bool   `json:"newtUpdateAvailable"`

	// Bandwidth stats
	MegabytesIn         float64 `json:"megabytesIn"`
	MegabytesOut        float64 `json:"megabytesOut"`
	LastBandwidthUpdate string  `json:"lastBandwidthUpdate"`

	// Advanced fields
	ExitNodeID          int    `json:"exitNodeId"`
	LastHolePunch       string `json:"lastHolePunch"`
	ListenPort          int    `json:"listenPort"`
	DockerSocketEnabled bool   `json:"dockerSocketEnabled"`
	RemoteSubnets       string `json:"remoteSubnets"`
}

// ResourceCreateSpec defines the specification for creating a resource
type ResourceCreateSpec struct {
	Name     string `json:"name"`
	HTTP     bool   `json:"http"`
	Protocol string `json:"protocol"`
	// HTTP-specific fields
	Subdomain   string `json:"subdomain,omitempty"`
	DomainID    string `json:"domainId,omitempty"`
	SSO         bool   `json:"sso,omitempty"`
	BlockAccess bool   `json:"blockAccess,omitempty"`
	// TCP/UDP-specific fields
	ProxyPort   int32 `json:"proxyPort,omitempty"`
	EnableProxy bool  `json:"enableProxy,omitempty"`
}

// ResourceUpdateSpec defines the specification for updating a resource
// Uses pointers to distinguish between "not set" and "set to false"
type ResourceUpdateSpec struct {
	SSO         *bool `json:"sso,omitempty"`
	BlockAccess *bool `json:"blockAccess,omitempty"`
	Enabled     *bool `json:"enabled,omitempty"`
}

// TargetCreateSpec defines the specification for creating a target
type TargetCreateSpec struct {
	IP            string `json:"ip"`
	Port          int32  `json:"port"`
	Method        string `json:"method"`
	Enabled       bool   `json:"enabled"`
	Path          string `json:"path,omitempty"`
	PathMatchType string `json:"pathMatchType,omitempty"`
	Priority      int32  `json:"priority,omitempty"`
}

// Resource represents a Pangolin resource
// The Integration API returns resourceId (numeric) on creation; keep both and normalize.
type Resource struct {
	ID         string `json:"id,omitempty"`         // may be empty in create response
	ResourceID int    `json:"resourceId,omitempty"` // present in create response
	Name       string `json:"name,omitempty"`
	SiteID     int    `json:"siteId,omitempty"`
	HTTP       bool   `json:"http"`
	Protocol   string `json:"protocol"`
	Subdomain  string `json:"subdomain,omitempty"`
	DomainID   string `json:"domainId,omitempty"`
	FullDomain string `json:"fullDomain,omitempty"`
	Enabled    bool   `json:"enabled"`
	SSO        bool   `json:"sso"`
	BlockAccess bool  `json:"blockAccess"`
}

// EffectiveID returns a string identifier usable in URL paths.
func (r *Resource) EffectiveID() string {
	if r.ID != "" {
		return r.ID
	}
	if r.ResourceID > 0 {
		return strconv.Itoa(r.ResourceID)
	}
	return ""
}

// Target represents a Pangolin target
type Target struct {
	ID       string `json:"id,omitempty"`
	TargetID int    `json:"targetId,omitempty"`
	SiteID   int    `json:"siteId,omitempty"`
	IP       string `json:"ip,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

// EffectiveID returns the target ID as a string
func (t *Target) EffectiveID() string {
	if t.ID != "" {
		return t.ID
	}
	if t.TargetID != 0 {
		return strconv.Itoa(t.TargetID)
	}
	return ""
}
