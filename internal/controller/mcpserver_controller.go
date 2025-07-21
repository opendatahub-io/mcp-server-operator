/*
Copyright 2025.

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

package controller

import (
	"context"
	"reflect"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	mcpserverv1 "github.com/opendatahub-io/mcp-server-operator/api/v1"
)

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mcpserver.opendatahub.io,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mcpserver.opendatahub.io,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mcpserver.opendatahub.io,resources=mcpservers/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=services,verbs=create;get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=create;get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=create;get;list;watch;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Create logger with passed in context value
	logger := logf.FromContext(ctx)

	// Creates an empty MCP server with no values inside.
	mcpServer := &mcpserverv1.MCPServer{}

	// creates a key used to identify the MCPServer with the name and namespace being used.
	ref := client.ObjectKey{Name: req.Name, Namespace: req.Namespace}
	// Gets the MCPServer instance using the context and previous key made to then fill up the mcpServer object
	err := r.Client.Get(ctx, ref, mcpServer)

	// If the error is not nil (or empty) then it returns an error and exits the function.
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Resource no longer exists – nothing to do.
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch MCPServer")
		return ctrl.Result{}, err

	}

	originalStatus := mcpServer.Status.DeepCopy()

	// Calls the reconcileMCPServerDeployment function, passing through the context, client and the mcpServer object
	err = r.reconcileMCPServerDeployment(ctx, r.Client, mcpServer)
	if err != nil {
		logger.Error(err, "Failed to reconcile MCPServer Deployment")
		return ctrl.Result{}, err
	}

	// Calls the reconcileMCPServerService function, passes through context, client and mcpserver object
	err = r.reconcileMCPServerService(ctx, r.Client, mcpServer)
	if err != nil {
		logger.Error(err, "Failed to reconcile MCPServer Service")
		return ctrl.Result{}, err
	}

	err = r.reconcileMCPServerRoute(ctx, r.Client, mcpServer)
	if err != nil {
		logger.Error(err, "Failed to reconcile MCPServer Route")
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&mcpServer.Status.Conditions, r.getDeploymentCondition(ctx, r.Client, mcpServer))
	meta.SetStatusCondition(&mcpServer.Status.Conditions, r.getServiceCondition(ctx, r.Client, mcpServer))
	meta.SetStatusCondition(&mcpServer.Status.Conditions, r.getRouteCondition(ctx, r.Client, mcpServer))

	overallReady := r.getOverallCondition(mcpServer)
	meta.SetStatusCondition(&mcpServer.Status.Conditions, overallReady)

	if !reflect.DeepEqual(originalStatus, &mcpServer.Status) {
		logger.Info("Status has changed, attempting to update")
		if err = r.Status().Update(ctx, mcpServer); err != nil {
			logger.Error(err, "unable to update MCPServer status")
			return ctrl.Result{}, err
		}
		logger.Info("Successfully updated MCPServer status")
	}

	if overallReady.Status != metav1.ConditionTrue {
		logger.Info("MCPServer not yet fully ready, re-queuing...", "reason", overallReady.Reason, "message", overallReady.Message)
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	logger.Info("MCPServer is fully ready", "name", mcpServer.Name, "namespace", mcpServer.Namespace)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a predicate to filter resources with the "opendatahub.io/mcp-server" label
	labelPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetLabels()["opendatahub.io/mcp-server"] != ""
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetLabels()["opendatahub.io/mcp-server"] != ""
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetLabels()["opendatahub.io/mcp-server"] != ""
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetLabels()["opendatahub.io/mcp-server"] != ""
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpserverv1.MCPServer{}).
		Watches(&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.mapResourceToMCPServer),
			builder.WithPredicates(labelPredicate)).
		Watches(&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.mapResourceToMCPServer),
			builder.WithPredicates(labelPredicate)).
		Watches(&routev1.Route{},
			handler.EnqueueRequestsFromMapFunc(r.mapResourceToMCPServer),
			builder.WithPredicates(labelPredicate)).
		Named("mcpserver").
		Complete(r)
}

// mapResourceToMCPServer maps a watched resource to the MCPServer that owns it
func (r *MCPServerReconciler) mapResourceToMCPServer(ctx context.Context, obj client.Object) []reconcile.Request {
	// Get the owner references to find the MCPServer that owns this resource
	for _, ownerRef := range obj.GetOwnerReferences() {
		if ownerRef.Kind == "MCPServer" && ownerRef.APIVersion == mcpserverv1.GroupVersion.String() {
			return []reconcile.Request{
				{
					NamespacedName: client.ObjectKey{
						Name:      ownerRef.Name,
						Namespace: obj.GetNamespace(),
					},
				},
			}
		}
	}

	// If no MCPServer owner found, try to find by matching names/labels
	// This is a fallback mechanism
	return []reconcile.Request{
		{
			NamespacedName: client.ObjectKey{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		},
	}
}
