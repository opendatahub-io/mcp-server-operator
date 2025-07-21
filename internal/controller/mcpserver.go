package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcpserverv1 "github.com/opendatahub-io/mcp-server-operator/api/v1"
)

const (
	mcpServerAppLabelKey = "opendatahub.io/mcp-server"

	// Condition types
	DeploymentAvailable = "DeploymentAvailable"
	RouteAvailable      = "RouteAvailable"
	ServiceAvailable    = "ServiceAvailable"
	OverallAvailable    = "Available"

	// Reason types
	ReasonNotFoundSuffix   = "NotFound"
	ReasonReadySuffix      = "Ready"
	ReasonNotReadySuffix   = "NotReady"
	ReasonGetFailedSuffix  = "GetFailed"
	ReasonRouteNotAdmitted = "RouteNotAdmitted"
)

func (r *MCPServerReconciler) reconcileMCPServerDeployment(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) error {

	labels := map[string]string{
		mcpServerAppLabelKey: cr.Name,
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: cr.Spec.Image,
						Name:  "mcp-server",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8000,
							Name:          "http",
						}},
						Command: []string{"./kubernetes-mcp-server"},
						Args:    []string{"--port", "8000", "--log-level", "9"},
					}},
				},
			},
		},
	}

	// Set the MCPServer to own the deployment.
	err := ctrl.SetControllerReference(cr, deployment, r.Scheme)
	if err != nil {
		return err
	}

	err = cli.Create(ctx, deployment)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (r *MCPServerReconciler) reconcileMCPServerService(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) error {

	labels := map[string]string{
		mcpServerAppLabelKey: cr.Name,
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8000,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Set MCPServer to own the service.
	err := ctrl.SetControllerReference(cr, service, r.Scheme)
	if err != nil {
		return err
	}

	err = cli.Create(ctx, service)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (r *MCPServerReconciler) reconcileMCPServerRoute(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) error {

	labels := map[string]string{
		mcpServerAppLabelKey: cr.Name,
	}

	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: cr.Name,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
		},
	}

	// Set MCPServer to own the route.
	err := ctrl.SetControllerReference(cr, route, r.Scheme)
	if err != nil {
		return err
	}

	err = cli.Create(ctx, route)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (r *MCPServerReconciler) getDeploymentCondition(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) metav1.Condition {
	dep := &appsv1.Deployment{}

	err := cli.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, dep)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Deployment %s cannot be found", cr.Name),
			}
		}
		return metav1.Condition{
			Type:    DeploymentAvailable,
			Status:  metav1.ConditionUnknown,
			Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonGetFailedSuffix),
			Message: fmt.Sprintf("Failed to retrieve Deployment %s, %v", cr.Name, err),
		}
	}

	// Converts the deployment's status conditions into a metav1 condition.
	// This is for future use in the isStatusConditionTrue call.
	var deploymentConditions = make([]metav1.Condition, 0)
	for _, cond := range dep.Status.Conditions {
		deploymentConditions = append(deploymentConditions, metav1.Condition{
			Type:    string(cond.Type),
			Status:  metav1.ConditionStatus(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	if !meta.IsStatusConditionTrue(deploymentConditions, string(appsv1.DeploymentAvailable)) {
		return metav1.Condition{
			Type:    DeploymentAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
			Message: fmt.Sprintf("Deployment %s is not yet available", cr.Name),
		}
	}

	return metav1.Condition{
		Type:    DeploymentAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonReadySuffix),
		Message: fmt.Sprintf("Deployment %s is available", cr.Name),
	}

}

func (r *MCPServerReconciler) getServiceCondition(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) metav1.Condition {

	svc := &corev1.Service{}
	err := cli.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, svc)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return metav1.Condition{
				Type:    ServiceAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Service %s not found", cr.Name),
			}
		}
		return metav1.Condition{
			Type:    ServiceAvailable,
			Status:  metav1.ConditionUnknown,
			Reason:  fmt.Sprintf("%s%s", "Service", ReasonGetFailedSuffix),
			Message: fmt.Sprintf("Failed to get Service %s: %v", cr.Name, err),
		}
	}

	return metav1.Condition{
		Type:    ServiceAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  fmt.Sprintf("%s%s", "Service", ReasonReadySuffix),
		Message: fmt.Sprintf("Service %s exists and is available", cr.Name),
	}
}

func (r *MCPServerReconciler) getRouteCondition(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) metav1.Condition {
	route := &routev1.Route{}
	err := cli.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, route)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Route %s not found", cr.Name),
			}
		}
		return metav1.Condition{
			Type:    RouteAvailable,
			Status:  metav1.ConditionUnknown,
			Reason:  fmt.Sprintf("%s%s", "Route", ReasonGetFailedSuffix),
			Message: fmt.Sprintf("Failed to get Route %s: %v", cr.Name, err),
		}
	}

	admitted := false
	for _, ingress := range route.Status.Ingress {
		for _, cond := range ingress.Conditions {
			if cond.Type == routev1.RouteAdmitted && cond.Status == corev1.ConditionTrue {
				admitted = true
				break
			}
		}
		if admitted {
			break
		}
	}

	if !admitted {
		return metav1.Condition{
			Type:    RouteAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonRouteNotAdmitted,
			Message: fmt.Sprintf("Route %s has not been admitted by a router yet", cr.Name),
		}
	}

	return metav1.Condition{
		Type:    RouteAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  fmt.Sprintf("%s%s", "Route", ReasonReadySuffix),
		Message: fmt.Sprintf("Route %s is admitted and active", cr.Name),
	}

}

func (r *MCPServerReconciler) getOverallCondition(cr *mcpserverv1.MCPServer) metav1.Condition {

	depCondition := meta.FindStatusCondition(cr.Status.Conditions, DeploymentAvailable)
	svcCondition := meta.FindStatusCondition(cr.Status.Conditions, ServiceAvailable)
	routeCondition := meta.FindStatusCondition(cr.Status.Conditions, RouteAvailable)

	if depCondition == nil || depCondition.Status != metav1.ConditionTrue {
		return metav1.Condition{
			Type:    OverallAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
			Message: "Deployment is not yet ready",
		}
	}
	if svcCondition == nil || svcCondition.Status != metav1.ConditionTrue {
		return metav1.Condition{
			Type:    OverallAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  fmt.Sprintf("%s%s", "Service", ReasonNotReadySuffix),
			Message: "Service is not yet ready",
		}
	}
	if routeCondition == nil || routeCondition.Status != metav1.ConditionTrue {
		return metav1.Condition{
			Type:    OverallAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  fmt.Sprintf("%s%s", "Route", ReasonNotReadySuffix),
			Message: "Route is not yet ready",
		}
	}

	return metav1.Condition{
		Type:    OverallAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  "AllComponentsReady",
		Message: "All managed components (Deployment, Service, Route) are ready",
	}

}
