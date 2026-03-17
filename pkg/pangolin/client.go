package pangolin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client represents a Pangolin API client for interacting with the Pangolin platform.
// It handles authentication, request construction, and response parsing for all API operations.
type Client struct {
	endpoint string       // Base API endpoint URL (e.g., "https://api.pangolin.dobryops.com")
	apiKey   string       // API key for authentication
	client   *http.Client // HTTP client with configured timeout
}

// NewClient creates a new Pangolin API client with the specified endpoint and API key.
//
// Parameters:
//   - endpoint: Base URL of the Pangolin API (e.g., "https://api.pangolin.dobryops.com")
//   - apiKey: API key for authentication (obtained from Pangolin dashboard)
//
// The client is configured with a 30-second timeout for all requests.
func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest constructs and executes an HTTP request to the Pangolin API.
//
// All requests are made to /v1/<path> with proper authentication headers.
// Request bodies are automatically JSON-encoded if provided.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - path: API path relative to /v1/ (e.g., "orgs", "org/my-org/sites")
//   - body: Request body to be JSON-encoded (nil for no body)
//
// Returns:
//   - HTTP response (caller must close response body)
//   - Error if request construction or execution fails
//
// Request Headers:
//   - Content-Type: application/json
//   - Authorization: Bearer <apiKey>
//   - User-Agent: pangolin-operator/1.0
func (c Client) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	cleanPath := strings.TrimLeft(path, "/")
	url := fmt.Sprintf("%s/v1/%s", strings.TrimRight(c.endpoint, "/"), cleanPath)

	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = b
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pangolin-operator/1.0")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	return c.client.Do(req)
}

// ListOrganizations retrieves all organizations accessible with the current API key.
//
// Returns:
//   - Slice of Organization objects with ID, name, and subnet information
//   - Error if request fails or API returns non-success status
//
// Used for:
//   - Organization validation during binding
//   - Organization discovery when no specific org is specified
//   - Listing available organizations for selection
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	resp, err := c.makeRequest(ctx, "GET", "/orgs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list orgs failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Orgs []Organization `json:"orgs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return result.Data.Orgs, nil
}

// ListDomains retrieves all domains configured for an organization.
//
// Domains are used for HTTP resource exposure, allowing resources to be accessed
// via subdomains (e.g., app.mydomain.com).
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to query domains for
//
// Returns:
//   - Slice of Domain objects with ID, base domain, verification status, and configuration
//   - Error if request fails or organization not found
//
// Domain Types:
//   - Verified: Domains that have been verified and are ready for use
//   - Unverified: Domains pending DNS verification
//   - Failed: Domains that failed verification
func (c *Client) ListDomains(ctx context.Context, orgID string) ([]Domain, error) {
	path := fmt.Sprintf("/org/%s/domains?limit=1000&offset=0", orgID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list domains failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Domains []Domain `json:"domains"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return result.Data.Domains, nil
}

// ListSites retrieves all sites (tunnel endpoints) for an organization.
//
// Sites represent physical or virtual locations where resources can be deployed.
// Each site has a tunnel client that connects to the Pangolin platform.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to query sites for
//
// Returns:
//   - Slice of Site objects with ID, name, type, and status information
//   - Error if request fails or organization not found
//
// Site Types:
//   - newt: Newt protocol tunnel (recommended)
//   - wireguard: WireGuard VPN tunnel
//   - other: Custom tunnel implementations
func (c *Client) ListSites(ctx context.Context, orgID string) ([]Site, error) {
	path := fmt.Sprintf("/org/%s/sites?limit=1000&offset=0", orgID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list sites failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Sites []Site `json:"sites"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return result.Data.Sites, nil
}

// GetSiteByID retrieves a specific site by its numeric site ID.
//
// This is the primary method for binding to existing sites when the numeric ID is known.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - siteID: Numeric site identifier
//
// Returns:
//   - Site object with complete site information
//   - Error if site not found or request fails
func (c *Client) GetSiteByID(ctx context.Context, siteID int) (*Site, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/site/%d", siteID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("get site by id failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    Site `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return &result.Data, nil
}

// GetSiteByNiceID retrieves a specific site by its human-readable nice ID.
//
// Nice IDs are automatically generated human-readable identifiers (e.g., "happy-brave-tiger").
// This method requires both organization ID and nice ID as nice IDs are only unique within an org.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID that owns the site
//   - niceID: Human-readable site identifier
//
// Returns:
//   - Site object with complete site information
//   - Error if site not found or request fails
func (c *Client) GetSiteByNiceID(ctx context.Context, orgID, niceID string) (*Site, error) {
	path := fmt.Sprintf("/org/%s/site/%s", orgID, niceID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("get site by niceId failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    Site `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return &result.Data, nil
}

// CreateSite creates a new site within an organization.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to create the site in
//   - name: Display name for the site
//   - siteType: Type of site/tunnel to create (e.g., "newt", "wireguard")
//
// Returns:
//   - Site object with the created site's information including assigned IDs
//   - Error if creation fails or site with same name already exists
//
// Site Creation:
//   - Generates unique numeric siteId
//   - Generates unique human-readable niceId
//   - Allocates network subnet for the site
//   - Returns tunnel credentials (for newt sites)
func (c *Client) CreateSite(ctx context.Context, orgID, name, siteType string) (*Site, error) {
	body := map[string]interface{}{
		"name": name,
		"type": siteType,
	}
	path := fmt.Sprintf("/org/%s/site", orgID)
	resp, err := c.makeRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("create site failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    Site `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return &result.Data, nil
}

// ListResources retrieves all resources for an organization.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to query resources for
//
// Returns:
//   - Slice of Resource objects
//   - Error if request fails
func (c *Client) ListResources(ctx context.Context, orgID string) ([]Resource, error) {
	path := fmt.Sprintf("/org/%s/resources?limit=1000&offset=0", orgID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list resources failed: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Resources []Resource `json:"resources"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	return result.Data.Resources, nil
}

// FindResourceBySubdomain finds a resource by its subdomain and domain ID.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to search in
//   - subdomain: Subdomain to match
//   - domainID: Domain ID to match (required for accurate matching)
//
// Returns:
//   - Resource if found, nil if not found
//   - Error if request fails
func (c *Client) FindResourceBySubdomain(ctx context.Context, orgID, subdomain, domainID string) (*Resource, error) {
	resources, err := c.ListResources(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Match by subdomain AND domainID (both required for multi-domain support)
	for _, r := range resources {
		if r.Subdomain == subdomain && r.DomainID == domainID {
			return &r, nil
		}
	}

	return nil, nil // Not found
}

// FindResourceByName finds a resource by its display name.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to search in
//   - name: Resource name to match (e.g., "namespace/ingress/host")
//
// Returns:
//   - Resource if found, nil if not found
//   - Error if request fails
func (c *Client) FindResourceByName(ctx context.Context, orgID, name string) (*Resource, error) {
	resources, err := c.ListResources(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.Name == name {
			return &r, nil
		}
	}

	return nil, nil // Not found
}

// CreateResource creates a new resource at the organization level.
//
// Resources represent services to be exposed through Pangolin. They are created at the
// organization level as abstract definitions, then linked to concrete backends via targets.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - orgID: Organization ID to create the resource in
//   - siteID: Site ID for target association (can be empty for HTTP resources)
//   - spec: Resource specification including name, protocol, and configuration
//
// Returns:
//   - Resource object with assigned resource ID
//   - Error if creation fails or invalid configuration provided
//
// Resource Types:
//
// HTTP Resources (spec.HTTP = true):
//   - Requires: subdomain, domainId
//   - Exposed via HTTPS at subdomain.domain
//   - Automatically provisions SSL certificates
//
// TCP/UDP Resources (spec.HTTP = false):
//   - Requires: proxyPort, enableProxy
//   - Exposed via TCP/UDP proxy
//   - No SSL termination
//
// Resource Architecture:
//   - Resource: Abstract definition at org level
//   - Target: Concrete backend at site level
//   - One resource can have multiple targets for load balancing
func (c *Client) CreateResource(ctx context.Context, orgID, siteID string, spec ResourceCreateSpec) (*Resource, error) {
	data := map[string]interface{}{
		"name":     spec.Name,
		"http":     spec.HTTP,
		"protocol": spec.Protocol,
	}

	// Add HTTP-specific fields
	if spec.HTTP {
		if spec.Subdomain != "" {
			data["subdomain"] = spec.Subdomain
		}
		if spec.DomainID != "" {
			data["domainId"] = spec.DomainID
		}
	} else {
		// Add TCP/UDP proxy fields
		if spec.ProxyPort > 0 {
			data["proxyPort"] = spec.ProxyPort
		}
		if spec.EnableProxy {
			data["enableProxy"] = spec.EnableProxy
		}
	}

	// Always use org-level endpoint for resource creation
	path := fmt.Sprintf("/org/%s/resource", orgID)

	resp, err := c.makeRequest(ctx, "PUT", path, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read full response for error handling
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate content type
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return nil, fmt.Errorf("unexpected content-type %q status %d: %s", ct, resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Success bool        `json:"success"`
		Data    Resource    `json:"data"`
		Error   interface{} `json:"error,omitempty"`
		Message string      `json:"message,omitempty"`
		Status  int         `json:"status,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(bodyBytes))
	}

	// Enhanced error handling with API message
	if !result.Success {
		errMsg := "API request was not successful"
		if result.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, result.Message)
		}
		if result.Status > 0 {
			errMsg = fmt.Sprintf("%s (status: %d)", errMsg, result.Status)
		}
		return nil, fmt.Errorf(errMsg)
	}

	// Normalize ID field (API may return either 'id' or 'resourceId')
	if result.Data.ID == "" && result.Data.ResourceID != 0 {
		result.Data.ID = strconv.Itoa(result.Data.ResourceID)
	}

	return &result.Data, nil
}

// CreateTarget creates a target (backend) for a resource at a specific site.
//
// Targets link abstract resources to concrete backend services. A resource can have
// multiple targets for load balancing or multi-site deployments.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - resourceID: Resource ID to create target for
//   - siteID: Site ID where the backend exists (associates target with site)
//   - spec: Target specification including IP, port, method, and enabled state
//
// Returns:
//   - Target object with assigned target ID
//   - Error if creation fails or duplicate target exists
//
// Target Specification:
//   - IP: Backend IP address (can be internal cluster IP)
//   - Port: Backend port number
//   - Method: Protocol method (http, https, tcp, udp)
//   - Enabled: Whether target should receive traffic
//   - SiteID: Associates target with specific site for routing
//
// Target Matching:
//   - Targets are unique per (IP, port, method, siteID) combination
//   - Creating duplicate target returns "already exists" error
//   - Use ListTargets to check for existing targets before creating
func (c *Client) CreateTarget(ctx context.Context, resourceID, siteID string, spec TargetCreateSpec) (*Target, error) {
	data := map[string]interface{}{
		"ip":      spec.IP,
		"port":    spec.Port,
		"method":  spec.Method,
		"enabled": spec.Enabled,
	}

	// Add siteID to associate target with site
	if siteID != "" {
		data["siteId"] = mustParseInt(siteID)
	}

	resp, err := c.makeRequest(ctx, "PUT", fmt.Sprintf("resource/%s/target", resourceID), data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read full response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate content type
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return nil, fmt.Errorf("unexpected content-type %q status %d: %s", ct, resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Success bool   `json:"success"`
		Data    Target `json:"data"`
		Message string `json:"message,omitempty"`
		Status  int    `json:"status,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode target response: %w, body: %s", err, string(bodyBytes))
	}

	if !result.Success {
		errMsg := "API request was not successful"
		if result.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, result.Message)
		}
		return nil, fmt.Errorf(errMsg)
	}

	return &result.Data, nil
}

// DeleteTarget deletes a target from a resource.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - resourceID: Resource ID that owns the target
//   - targetID: Target ID to delete
//
// Returns error if deletion fails.
func (c *Client) DeleteTarget(ctx context.Context, resourceID, targetID string) error {
	path := fmt.Sprintf("resource/%s/target/%s", resourceID, targetID)
	resp, err := c.makeRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("delete target failed: status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// ListTargets retrieves all targets (backends) for a specific resource.
//
// Used for:
//   - Checking if target already exists before creation (idempotency)
//   - Discovering all backends for a resource
//   - Tracking manually added targets via UI
//   - Multi-target management and load balancing
//
// Parameters:
//   - ctx: Context for request cancellation
//   - resourceID: Resource ID to query targets for
//
// Returns:
//   - Slice of Target objects with ID, IP, port, method, and site association
//   - Error if request fails or resource not found
//
// Target List Uses:
//   - Idempotency: Check before creating to avoid duplicates
//   - Discovery: Find all backends including manually added ones
//   - Monitoring: Track target health and availability
func (c *Client) ListTargets(ctx context.Context, resourceID string) ([]Target, error) {
	path := fmt.Sprintf("resource/%s/targets?limit=1000&offset=0", resourceID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return nil, fmt.Errorf("unexpected content-type %q status %d: %s", ct, resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Targets []Target `json:"targets"`
		} `json:"data"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(bodyBytes))
	}

	if !result.Success {
		errMsg := "API request was not successful"
		if result.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, result.Message)
		}
		return nil, fmt.Errorf(errMsg)
	}

	return result.Data.Targets, nil
}

// UpdateResource updates an existing resource's configuration.
//
// This method is used to update resource settings that cannot be set during creation,
// such as SSO authentication settings.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - resourceID: Resource ID to update
//   - spec: Update specification with fields to modify
//
// Returns:
//   - Updated Resource object
//   - Error if update fails or resource not found
//
// Updatable Fields:
//   - SSO: Enable/disable SSO authentication
//   - BlockAccess: Block access until authenticated (requires SSO enabled)
//   - Name, Subdomain, Enabled, etc.
func (c *Client) UpdateResource(ctx context.Context, resourceID string, spec ResourceUpdateSpec) (*Resource, error) {
	data := make(map[string]interface{})

	// Only include fields that are explicitly set
	if spec.SSO != nil {
		data["sso"] = *spec.SSO
	}
	if spec.BlockAccess != nil {
		data["blockAccess"] = *spec.BlockAccess
	}
	if spec.Enabled != nil {
		data["enabled"] = *spec.Enabled
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	path := fmt.Sprintf("/resource/%s", resourceID)
	resp, err := c.makeRequest(ctx, "POST", path, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return nil, fmt.Errorf("unexpected content-type %q status %d: %s", ct, resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Success bool        `json:"success"`
		Data    Resource    `json:"data"`
		Error   interface{} `json:"error,omitempty"`
		Message string      `json:"message,omitempty"`
		Status  int         `json:"status,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(bodyBytes))
	}

	if !result.Success {
		errMsg := "API request was not successful"
		if result.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, result.Message)
		}
		if result.Status > 0 {
			errMsg = fmt.Errorf("%s (status: %d)", errMsg, result.Status).Error()
		}
		return nil, fmt.Errorf(errMsg)
	}

	return &result.Data, nil
}

// mustParseInt converts a string to an integer, returning 0 on error.
// Used for parsing site IDs and other numeric identifiers from string format.
func mustParseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
