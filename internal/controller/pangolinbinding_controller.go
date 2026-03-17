package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tunnelv1alpha1 "github.com/bovf/pangolin-operator/api/v1alpha1"
)

const (
	BindingFinalizerName = "binding.pangolin.io/finalizer"
)

// PangolinBindingReconciler reconciles a PangolinBinding object
type PangolinBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolinbindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolinbindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolinbindings/finalizers,verbs=update
//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolinresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolintunnels,verbs=get;list;watch
//+kubebuilder:rbac:groups=tunnel.pangolin.io,resources=pangolinorganizations,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch

// Reconcile implements the reconciliation logic for PangolinBinding.
//
// PangolinBinding provides a high-level abstraction for exposing Kubernetes Services
// through Pangolin tunnels. It automatically creates and manages the underlying
// PangolinResource and keeps it synchronized with the Service.
//
// Reconciliation Flow:
//  1. Fetch PangolinBinding from Kubernetes
//  2. Handle deletion if binding is being deleted
//  3. Add finalizer if not present
//  4. Fetch referenced Service to get ClusterIP
//  5. Get organization and wait for it to be ready
//  6. Ensure tunnel exists (use specified tunnel or create default)
//  7. Wait for tunnel to be ready
//  8. Create or update PangolinResource with Service ClusterIP as target
//  9. Wait for resource to be ready
//  10. Optionally update service endpoints for multi-pod services
//  11. Update binding status with generated resource name and URL
//
// Key Features:
//   - Automatic PangolinResource creation from Service
//   - Service ClusterIP used as target backend
//   - Owner references ensure resource cleanup on binding deletion
//   - Auto-update of endpoints for multi-pod services (optional)
//   - Status reflects generated resource and access URL
//
// Example Usage:
//
//	A PangolinBinding for a Service named "myapp" on port 8080 will:
//	- Create a PangolinResource named "myapp-binding"
//	- Configure target as Service ClusterIP:8080
//	- Expose via Pangolin with subdomain from binding spec
//	- Update binding status with full access URL
func (r *PangolinBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the PangolinBinding instance
	binding := &tunnelv1alpha1.PangolinBinding{}
	err := r.Get(ctx, req.NamespacedName, binding)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PangolinBinding resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PangolinBinding")
		return ctrl.Result{}, err
	}

	// Handle deletion if binding is being deleted
	if binding.DeletionTimestamp != nil {
		return r.handleBindingDeletion(ctx, binding)
	}

	// Add finalizer to ensure proper cleanup
	if !controllerutil.ContainsFinalizer(binding, BindingFinalizerName) {
		controllerutil.AddFinalizer(binding, BindingFinalizerName)
		return ctrl.Result{}, r.Update(ctx, binding)
	}

	// Get the referenced service to extract ClusterIP and validate it exists
	service, err := r.getServiceForBinding(ctx, binding)
	if err != nil {
		logger.Error(err, "Failed to get referenced service")
		return r.updateBindingStatus(ctx, binding, "Error", err.Error())
	}

	// Get the referenced organization for API credentials
	org, err := r.getOrganizationForBinding(ctx, binding)
	if err != nil {
		logger.Error(err, "Failed to get referenced organization")
		return r.updateBindingStatus(ctx, binding, "Error", err.Error())
	}

	// Wait for organization to be ready
	if org.Status.Status != "Ready" {
		logger.Info("Organization not ready yet, waiting", "organization", org.Name)
		return r.updateBindingStatus(ctx, binding, "Waiting", "Waiting for organization to be ready")
	}

	// Get or create tunnel for the binding
	tunnel, err := r.ensureTunnelForBinding(ctx, binding, org)
	if err != nil {
		logger.Error(err, "Failed to ensure tunnel for binding")
		return r.updateBindingStatus(ctx, binding, "Error", err.Error())
	}

	// Wait for tunnel to be ready
	if tunnel.Status.Status != "Ready" {
		logger.Info("Tunnel not ready yet, waiting", "tunnel", tunnel.Name)
		return r.updateBindingStatus(ctx, binding, "Waiting", "Waiting for tunnel to be ready")
	}

	// Create or update PangolinResource based on binding spec and service
	resource, err := r.reconcileResourceForBinding(ctx, binding, tunnel, service)
	if err != nil {
		logger.Error(err, "Failed to reconcile resource for binding")
		return r.updateBindingStatus(ctx, binding, "Error", err.Error())
	}

	// Wait for resource to be ready
	if resource.Status.Status != "Ready" {
		logger.Info("Resource not ready yet, waiting", "resource", resource.Name)
		return r.updateBindingStatus(ctx, binding, "Waiting", "Waiting for resource to be ready")
	}

	// Update service endpoints if auto-update is enabled
	// This is useful for multi-pod services where endpoints change dynamically
	if binding.Spec.AutoUpdateTargets == nil || *binding.Spec.AutoUpdateTargets {
		err = r.updateServiceEndpoints(ctx, binding, service)
		if err != nil {
			logger.Error(err, "Failed to update service endpoints")
			// Don't fail the reconciliation for endpoint update errors
		}
	}

	// Update binding status with generated resource information
	binding.Status.GeneratedResourceName = resource.Name
	binding.Status.URL = resource.Status.URL
	binding.Status.ProxyEndpoint = resource.Status.ProxyEndpoint

	return r.updateBindingStatus(ctx, binding, "Ready", "Binding is ready")
}

// getServiceForBinding retrieves the Kubernetes Service referenced by the binding.
// The Service's ClusterIP will be used as the target for the PangolinResource.
//
// Returns an error if:
//   - Service does not exist
//   - Service is not in the specified namespace
func (r *PangolinBindingReconciler) getServiceForBinding(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: binding.Spec.ServiceRef.Namespace,
		Name:      binding.Spec.ServiceRef.Name,
	}, service)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s/%s: %w", binding.Spec.ServiceRef.Namespace, binding.Spec.ServiceRef.Name, err)
	}
	return service, nil
}

// getOrganizationForBinding retrieves the PangolinOrganization referenced by the binding.
// The organization provides API credentials and domain configuration.
func (r *PangolinBindingReconciler) getOrganizationForBinding(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding) (*tunnelv1alpha1.PangolinOrganization, error) {
	org := &tunnelv1alpha1.PangolinOrganization{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: binding.Namespace,
		Name:      binding.Spec.OrganizationRef.Name,
	}, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %s: %w", binding.Spec.OrganizationRef.Name, err)
	}
	return org, nil
}

// ensureTunnelForBinding gets the referenced tunnel or creates a default one.
//
// Tunnel Resolution:
//   - If spec.tunnelRef is set: Use the specified tunnel
//   - If spec.tunnelRef is nil: Create or find default tunnel for organization (TODO)
//
// Currently, a tunnel reference is required. Automatic tunnel creation is a future enhancement.
func (r *PangolinBindingReconciler) ensureTunnelForBinding(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding, org *tunnelv1alpha1.PangolinOrganization) (*tunnelv1alpha1.PangolinTunnel, error) {
	if binding.Spec.TunnelRef != nil {
		// Use specified tunnel
		tunnel := &tunnelv1alpha1.PangolinTunnel{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: binding.Namespace,
			Name:      binding.Spec.TunnelRef.Name,
		}, tunnel)
		if err != nil {
			return nil, fmt.Errorf("failed to get tunnel %s: %w", binding.Spec.TunnelRef.Name, err)
		}
		return tunnel, nil
	}

	// TODO: Create or find default tunnel for the organization
	// This would involve:
	// 1. Looking for a tunnel with label like "default-for-org=<orgName>"
	// 2. If not found, creating a new tunnel with automatic naming
	// 3. Binding it to an available site or creating a new site
	return nil, fmt.Errorf("tunnelRef is required - automatic tunnel creation not yet implemented")
}

// reconcileResourceForBinding creates or updates the PangolinResource managed by this binding.
//
// Resource Creation:
//   - Resource name: "<binding-name>-binding"
//   - Owner reference: Set to binding (ensures automatic deletion)
//   - Target IP: Service ClusterIP
//   - Target Port: From binding.spec.servicePort
//   - Protocol: From binding.spec.protocol
//   - HTTP/Proxy config: Copied from binding spec
//
// The resource is owned by the binding, so deleting the binding will automatically
// delete the resource due to Kubernetes garbage collection.
func (r *PangolinBindingReconciler) reconcileResourceForBinding(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding, tunnel *tunnelv1alpha1.PangolinTunnel, service *corev1.Service) (*tunnelv1alpha1.PangolinResource, error) {

	// Generate resource name from binding name
	resourceName := fmt.Sprintf("%s-binding", binding.Name)

	// Check if resource already exists
	resource := &tunnelv1alpha1.PangolinResource{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: binding.Namespace,
		Name:      resourceName,
	}, resource)

	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new resource with owner reference to binding
		resource = &tunnelv1alpha1.PangolinResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: binding.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: binding.APIVersion,
						Kind:       binding.Kind,
						Name:       binding.Name,
						UID:        binding.UID,
						Controller: &[]bool{true}[0],
					},
				},
			},
			Spec: tunnelv1alpha1.PangolinResourceSpec{
				TunnelRef: tunnelv1alpha1.LocalObjectReference{
					Name: tunnel.Name,
				},
				Name:     fmt.Sprintf("%s-%s", binding.Spec.ServiceRef.Name, binding.Spec.Protocol),
				Protocol: binding.Spec.Protocol,
				Targets: []tunnelv1alpha1.TargetConfig{
					{
						IP:     service.Spec.ClusterIP,
						Port:   binding.Spec.ServicePort,
						Method: "http", // TODO: Derive from protocol (http/https/tcp)
					},
				},
			},
		}

		// Set HTTP or proxy configuration based on protocol
		if binding.Spec.Protocol == "http" && binding.Spec.HTTPConfig != nil {
			resource.Spec.HTTPConfig = binding.Spec.HTTPConfig
		} else if binding.Spec.ProxyConfig != nil {
			resource.Spec.ProxyConfig = binding.Spec.ProxyConfig
		}

		err = r.Create(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}
	}

	return resource, nil
}

// updateServiceEndpoints updates the target endpoints based on service endpoints.
//
// For multi-pod services, this tracks all pod IPs backing the service and updates
// the binding status with the current endpoint list. This is useful for monitoring
// and future multi-target support.
//
// Auto-Update Behavior:
//   - If spec.autoUpdateTargets is true (default): Continuously track endpoints
//   - If spec.autoUpdateTargets is false: Skip endpoint updates
//
// Current Implementation:
//   - Tracks endpoint IPs in binding status
//   - TODO: Update PangolinResource with multiple targets for load balancing
func (r *PangolinBindingReconciler) updateServiceEndpoints(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding, service *corev1.Service) error {
	// Get endpoints for the service
	endpoints := &corev1.Endpoints{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}, endpoints)
	if err != nil {
		return fmt.Errorf("failed to get endpoints: %w", err)
	}

	// Extract all endpoint addresses from all subsets
	var endpointAddresses []string
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			endpointAddresses = append(endpointAddresses, address.IP)
		}
	}

	// Update binding status with current endpoints
	binding.Status.ServiceEndpoints = endpointAddresses

	// TODO: Update PangolinResource to create multiple targets (one per endpoint)
	// This would enable true load balancing across all pod replicas
	// For now, we only use the Service ClusterIP as a single target

	return nil
}

// updateBindingStatus updates the status of a PangolinBinding with the given status and message.
//
// Status values:
//   - "Ready": Binding is active, resource is exposed
//   - "Error": Reconciliation encountered an error
//   - "Waiting": Waiting for dependencies (service, org, tunnel, resource)
//
// The function also updates the Ready condition with appropriate reason and message.
// If status is not "Ready", the reconcile will be requeued after 1 minute.
func (r *PangolinBindingReconciler) updateBindingStatus(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding, status, message string) (ctrl.Result, error) {
	binding.Status.Status = status
	binding.Status.ObservedGeneration = binding.Generation

	// Create Ready condition
	conditionType := "Ready"
	conditionStatus := metav1.ConditionTrue
	reason := "ReconcileSuccess"

	if status != "Ready" {
		conditionStatus = metav1.ConditionFalse
		reason = "ReconcileError"
	}

	now := metav1.NewTime(time.Now())
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: binding.Generation,
	}

	// Update existing condition or append new one
	conditionUpdated := false
	for i, condition := range binding.Status.Conditions {
		if condition.Type == conditionType {
			// Preserve LastTransitionTime if status hasn't changed
			if condition.Status != conditionStatus {
				newCondition.LastTransitionTime = now
			} else {
				newCondition.LastTransitionTime = condition.LastTransitionTime
			}
			binding.Status.Conditions[i] = newCondition
			conditionUpdated = true
			break
		}
	}

	if !conditionUpdated {
		binding.Status.Conditions = append(binding.Status.Conditions, newCondition)
	}

	// Update status
	err := r.Status().Update(ctx, binding)
	if status != "Ready" {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	return ctrl.Result{}, err
}

// handleBindingDeletion handles cleanup when a PangolinBinding is being deleted.
//
// Cleanup Process:
//   - The owned PangolinResource is automatically deleted due to owner reference
//   - No manual API cleanup is performed (resource remains in Pangolin)
//   - Finalizer is removed to allow binding deletion
//
// Future Enhancement:
//   - Could optionally delete the Pangolin resource via API
//   - Could provide a spec field to control cleanup behavior
func (r *PangolinBindingReconciler) handleBindingDeletion(ctx context.Context, binding *tunnelv1alpha1.PangolinBinding) (ctrl.Result, error) {
	// The owned PangolinResource will be automatically deleted due to owner reference
	// Kubernetes garbage collection handles this automatically
	controllerutil.RemoveFinalizer(binding, BindingFinalizerName)
	return ctrl.Result{}, r.Update(ctx, binding)
}

// SetupWithManager sets up the controller with the Manager.
//
// Controller Configuration:
//   - Watches PangolinBinding resources for changes
//   - Owns PangolinResource (will reconcile when owned resource changes)
//   - Does not watch Services or Tunnels directly (manual triggers required)
func (r *PangolinBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tunnelv1alpha1.PangolinBinding{}).
		Owns(&tunnelv1alpha1.PangolinResource{}).
		Complete(r)
}
