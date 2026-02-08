package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		nsName := "api"

		ns, err := corev1.NewNamespace(ctx, nsName, &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(nsName),
			},
		})
		if err != nil {
			return err
		}

		appLabels := pulumi.StringMap{
			"app": pulumi.String("nginx"),
		}

		deployment, err := appsv1.NewDeployment(ctx, nsName+"-nginx", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("nginx"),
				Namespace: pulumi.String(nsName),
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: appLabels,
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: appLabels,
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String("nginx"),
								Image: pulumi.String("nginx:latest"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{ns}))
		if err != nil {
			return err
		}

		svc, err := corev1.NewService(ctx, nsName+"-svc", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("nginx"),
				Namespace: pulumi.String(nsName),
			},
			Spec: &corev1.ServiceSpecArgs{
				Type:     pulumi.String("ClusterIP"),
				Selector: appLabels,
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(80),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{ns}))
		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(ctx, nsName+"-route", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("nginx"),
				Namespace: pulumi.String(nsName),
			},
			OtherFields: map[string]interface{}{
				"spec": pulumi.Map{
					"hostnames": pulumi.StringArray{
						pulumi.String("api.mirrorboards.network"),
					},
					"parentRefs": pulumi.Array{
						pulumi.Map{
							"name":        pulumi.String("mirrorboards-platform-gateway"),
							"namespace":   pulumi.String("aks-istio-ingress"),
							"sectionName": pulumi.String("https"),
						},
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": pulumi.String("nginx"),
									"port": pulumi.Int(80),
								},
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{deployment, svc}))
		if err != nil {
			return err
		}

		ctx.Export("Namespace", pulumi.String(nsName))

		return nil
	})
}
