package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mcpserverv1 "github.com/opendatahub-io/mcp-server-operator/api/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mcpServerName  = "test-mcpserver"
	testNamespace  = "test-namespace"
	mcpServerImage = "test-image"
)

var (
	CustomMCPDeploymentCommand = []string{"/bin/sh"}
	CustomMCPDeploymentArgs    = []string{"-c", "echo 'custom'"}
)

func TestMCPServerReconciler_reconcileMCPServerDeployment(t *testing.T) {
	// Create an existing deployment
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "mcp-server"}},
				},
			},
		},
	}

	objects := []runtime.Object{existingDeployment}

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}
	mcpServerWithCustoms := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image:   mcpServerImage,
			Command: CustomMCPDeploymentCommand,
			Args:    CustomMCPDeploymentArgs,
		},
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		wantCommand []string
		wantArgs    []string
	}{
		{
			name: "Verify MCPServer Deployment can be created with default values",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().Build(),
				cr:  mcpServer,
			},
			wantErr:     false,
			wantCommand: DefaultMCPDeploymentCommand,
			wantArgs:    DefaultMCPDeploymentArgs,
		},
		{
			name: "Verify if deployment exists the function does not return an error",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects(objects...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects(objects...).Build(),
				cr:  mcpServer,
			},
			wantErr: false,
		},
		{
			name: "Verify Deployment is created with custom command and args",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().Build(),
				cr:  mcpServerWithCustoms,
			},
			wantErr:     false,
			wantCommand: CustomMCPDeploymentCommand, // Expect the custom value
			wantArgs:    CustomMCPDeploymentArgs,    // Expect the custom value
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}

			err := r.reconcileMCPServerDeployment(context.Background(), tt.fields.Client, tt.args.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileMCPServerDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Fetch the reconciled deployment to verify its state
			foundDeployment := &appsv1.Deployment{}
			err = tt.fields.Client.Get(context.Background(), types.NamespacedName{Name: tt.args.cr.Name, Namespace: tt.args.cr.Namespace}, foundDeployment)
			if err != nil {
				t.Errorf("failed to get deployment for verification: %v", err)
			}

			// Verify the container's command and args
			container := foundDeployment.Spec.Template.Spec.Containers[0]
			if !reflect.DeepEqual(container.Command, tt.wantCommand) {
				t.Errorf("Command mismatch: got %v, want %v", container.Command, tt.wantCommand)
			}
			if !reflect.DeepEqual(container.Args, tt.wantArgs) {
				t.Errorf("Args mismatch: got %v, want %v", container.Args, tt.wantArgs)
			}
		})
	}
}

func TestMCPServerReconciler_reconcileMCPServerService(t *testing.T) {
	// Create an existing service
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
	}
	objects := []runtime.Object{existingService}

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Verify MCPServer Service can be created",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().Build(),
				cr:  mcpServer,
			},
			wantErr: false,
		},
		{
			name: "Verify if service exists the function does not return an error",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects(objects...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects(objects...).Build(),
				cr:  mcpServer,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if err := r.reconcileMCPServerService(tt.args.ctx, tt.args.cli, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("reconcileMCPServerService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPServerReconciler_reconcileMCPServerRoute(t *testing.T) {
	// Create a fake scheme
	fakeScheme := runtime.NewScheme()

	// Create an existing route
	existingRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
	}
	objects := []runtime.Object{existingRoute}

	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}
	err = routev1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add routev1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Verify MCPServer Route can be created",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).Build(),
				cr:  mcpServer,
			},
			wantErr: false,
		},
		{
			name: "Verify if route exists the function does not return an error",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects(objects...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects(objects...).Build(),
				cr:  mcpServer,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if err := r.reconcileMCPServerRoute(tt.args.ctx, tt.args.cli, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("reconcileMCPServerRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockErrorClient struct {
	client.Client
	errOnGet bool
	getError error
}

func (m *mockErrorClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if m.errOnGet {
		return m.getError
	}

	return m.Client.Get(ctx, key, obj, opts...)
}

func TestMCPServerReconciler_getDeploymentCondition(t *testing.T) {

	// Create a deployment with missing status
	deploymentWithoutStatus := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
	}

	// Create a deployment that is ready
	readyDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
					Reason: fmt.Sprintf("%s%s", "Deployment", ReasonReadySuffix),
				},
			},
		},
	}

	// Create a deployment with unready status conditions.
	unreadyDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionFalse,
					Reason: fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
				},
			},
		},
	}

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}

	mockGetError := fmt.Errorf("failed to get object")

	fakeErrorClient := &mockErrorClient{
		Client:   fake.NewClientBuilder().Build(),
		errOnGet: true,
		getError: mockGetError,
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   metav1.Condition
	}{
		{
			name: "Verify that if deployment isn't found, the DeploymentNotFound condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Deployment %s cannot be found", mcpServer.Name),
			},
		},
		{
			name: "Verify if the deployment get fails, the DeploymentGetFailed condition is returned",
			fields: fields{
				Client: fakeErrorClient,
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fakeErrorClient,
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionUnknown,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonGetFailedSuffix),
				Message: fmt.Sprintf("Failed to retrieve Deployment %s, %v", mcpServer.Name, mockGetError),
			},
		},
		{
			name: "Verify that if the deployment status is false, the DeploymentNotReady condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{unreadyDeployment}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{unreadyDeployment}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
				Message: fmt.Sprintf("Deployment %s is not yet available", mcpServer.Name),
			},
		},
		{
			name: "Verify that if deployment's status is missing, function returns DeploymentNotReady",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{deploymentWithoutStatus}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{deploymentWithoutStatus}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
				Message: fmt.Sprintf("Deployment %s is not yet available", mcpServer.Name),
			},
		},
		{
			name: "Verify that if deployment exists and the deployment is ready, the DeploymentReady condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{readyDeployment}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{readyDeployment}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    DeploymentAvailable,
				Status:  metav1.ConditionTrue,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonReadySuffix),
				Message: fmt.Sprintf("Deployment %s is available", mcpServer.Name),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if got := r.getDeploymentCondition(tt.args.ctx, tt.args.cli, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDeploymentCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPServerReconciler_getServiceCondition(t *testing.T) {

	// Create an existing service
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
	}

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}

	mockGetError := fmt.Errorf("mock get error")

	fakeErrorClient := &mockErrorClient{
		Client:   fake.NewClientBuilder().Build(),
		errOnGet: true,
		getError: mockGetError,
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   metav1.Condition
	}{
		{
			name: "Verify that if service isn't found, the ServiceNotFound condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    ServiceAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Service %s not found", mcpServer.Name),
			},
		},
		{
			name: "Verify if the service get fails, the ServiceNotFound condition is returned",
			fields: fields{
				Client: fakeErrorClient,
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fakeErrorClient,
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    ServiceAvailable,
				Status:  metav1.ConditionUnknown,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonGetFailedSuffix),
				Message: fmt.Sprintf("Failed to get Service %s: %v", mcpServer.Name, mockGetError),
			},
		},
		{
			name: "Verify that if service exists, the ServiceExists condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{existingService}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithRuntimeObjects([]runtime.Object{existingService}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    ServiceAvailable,
				Status:  metav1.ConditionTrue,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonReadySuffix),
				Message: fmt.Sprintf("Service %s exists and is available", mcpServer.Name),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if got := r.getServiceCondition(tt.args.ctx, tt.args.cli, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getServiceCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPServerReconciler_getRouteCondition(t *testing.T) {

	// Create a route that is not admitted
	nonAdmittedRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Status: routev1.RouteStatus{
			Ingress: []routev1.RouteIngress{
				{
					Conditions: []routev1.RouteIngressCondition{
						{
							Type:   routev1.RouteAdmitted,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
		},
	}

	// Create a route that has all conditions set to true, and thus admitted
	admittedRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Status: routev1.RouteStatus{
			Ingress: []routev1.RouteIngress{
				{
					Conditions: []routev1.RouteIngressCondition{
						{
							Type:   routev1.RouteAdmitted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}

	// Create a route with no existing ingress
	missingIngressRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
	}

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}
	err = routev1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add routev1 scheme: %v", err)
	}

	// Create context
	testContext := context.Background()

	mcpServer := &mcpserverv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpServerName,
			Namespace: testNamespace,
		},
		Spec: mcpserverv1.MCPServerSpec{
			Image: mcpServerImage,
		},
	}

	mockGetError := fmt.Errorf("mock get error")

	// Create a client with a fake error
	fakeErrorClient := &mockErrorClient{
		Client:   fake.NewClientBuilder().WithScheme(fakeScheme).Build(),
		errOnGet: true,
		getError: mockGetError,
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		cli client.Client
		cr  *mcpserverv1.MCPServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   metav1.Condition
	}{
		{
			name: "Verify that if the route isn't found, the RouteNotFound condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonNotFoundSuffix),
				Message: fmt.Sprintf("Route %s not found", mcpServer.Name),
			},
		},
		{
			name: "Verify if the route get fails, the RouteGetFailed condition is returned",
			fields: fields{
				Client: fakeErrorClient,
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fakeErrorClient,
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionUnknown,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonGetFailedSuffix),
				Message: fmt.Sprintf("Failed to get Route %s: %v", mcpServer.Name, mockGetError),
			},
		},
		{
			name: "Verify that if the RouteAdmitted condition is not true, the RouteNotAdmitted condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{nonAdmittedRoute}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{nonAdmittedRoute}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  ReasonRouteNotAdmitted,
				Message: fmt.Sprintf("Route %s has not been admitted by a router yet", mcpServer.Name),
			},
		},
		{
			name: "Verify that if route's ingress is missing, function returns RouteNotAdmitted.",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{missingIngressRoute}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{missingIngressRoute}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  ReasonRouteNotAdmitted,
				Message: fmt.Sprintf("Route %s has not been admitted by a router yet", mcpServer.Name),
			},
		},
		{
			name: "Verify that if route is admitted, the RouteAdmitted condition is returned",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{admittedRoute}...).Build(),
				Scheme: fakeScheme,
			},
			args: args{
				ctx: testContext,
				cli: fake.NewClientBuilder().WithScheme(fakeScheme).WithRuntimeObjects([]runtime.Object{admittedRoute}...).Build(),
				cr:  mcpServer,
			},
			want: metav1.Condition{
				Type:    RouteAvailable,
				Status:  metav1.ConditionTrue,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonReadySuffix),
				Message: fmt.Sprintf("Route %s is admitted and active", mcpServer.Name),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if got := r.getRouteCondition(tt.args.ctx, tt.args.cli, tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRouteCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPServerReconciler_getOverallCondition(t *testing.T) {

	// Create a fake client with no existing resources
	fakeClient := fake.NewClientBuilder().Build()

	// Create a fake scheme
	fakeScheme := runtime.NewScheme()
	err := mcpserverv1.AddToScheme(fakeScheme)
	if err != nil {
		t.Errorf("failed to add mcpserverv1 scheme: %v", err)
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		cr *mcpserverv1.MCPServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   metav1.Condition
	}{
		{
			name: "Verify that if all components are ready, then the AllComponentsReady condition is returned.",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionTrue},
							{Type: ServiceAvailable, Status: metav1.ConditionTrue},
							{Type: RouteAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionTrue,
				Reason:  "AllComponentsReady",
				Message: "All managed components (Deployment, Service, Route) are ready",
			},
		},
		{
			name: "Verify that if depCondition is not true, the function returns the DeploymentNotReady condition",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionFalse},
							{Type: ServiceAvailable, Status: metav1.ConditionTrue},
							{Type: RouteAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
				Message: "Deployment is not yet ready",
			},
		},
		{
			name: "Verify that if svcCondition is not true, the function returns the ServiceNotReady condition.",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionTrue},
							{Type: ServiceAvailable, Status: metav1.ConditionFalse},
							{Type: RouteAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonNotReadySuffix),
				Message: "Service is not yet ready",
			},
		},

		{
			name: "Verify that if routeCondition isn't true, the function returns the RouteNotReady condition.",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionTrue},
							{Type: ServiceAvailable, Status: metav1.ConditionTrue},
							{Type: RouteAvailable, Status: metav1.ConditionFalse},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonNotReadySuffix),
				Message: "Route is not yet ready",
			},
		},
		{
			name: "Verify if the depCondition is nil, the function returns the DeploymentNotReady condition",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: ServiceAvailable, Status: metav1.ConditionTrue},
							{Type: RouteAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Deployment", ReasonNotReadySuffix),
				Message: "Deployment is not yet ready",
			},
		},
		{
			name: "Verify if the svcCondition is nil, the function returns the ServiceNotReady condition",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionTrue},
							{Type: RouteAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Service", ReasonNotReadySuffix),
				Message: "Service is not yet ready",
			},
		},
		{
			name: "Verify if the routeCondition is nil, the function returns the RouteNotReady condition",
			fields: fields{
				Client: fakeClient,
				Scheme: fakeScheme,
			},
			args: args{
				cr: &mcpserverv1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mcpServerName,
						Namespace: testNamespace,
					},
					Status: mcpserverv1.MCPServerStatus{
						Conditions: []metav1.Condition{
							{Type: DeploymentAvailable, Status: metav1.ConditionTrue},
							{Type: ServiceAvailable, Status: metav1.ConditionTrue},
						},
					},
					Spec: mcpserverv1.MCPServerSpec{
						Image: mcpServerImage,
					},
				},
			},
			want: metav1.Condition{
				Type:    OverallAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  fmt.Sprintf("%s%s", "Route", ReasonNotReadySuffix),
				Message: "Route is not yet ready",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MCPServerReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if got := r.getOverallCondition(tt.args.cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOverallCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
