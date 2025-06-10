package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcpserverv1 "github.com/opendatahub-io/mcp-server-operator/api/v1"
)

const mcpServerAppLabelKey = "mcp-server"

func (r *MCPServerReconciler) reconcileMCPServerDeployment(ctx context.Context, cli client.Client, cr *mcpserverv1.MCPServer) error {

	labels := map[string]string{
		mcpServerAppLabelKey: cr.Name,
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cr.APIVersion,
					Kind:       cr.Kind,
					Name:       cr.Name,
					UID:        cr.UID,
				},
			},
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
						Args:    []string{"--sse-port", "8000"},
					}},
				},
			},
		},
	}
	err := cli.Create(ctx, dep)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}
	return nil
}
