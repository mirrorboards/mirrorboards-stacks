package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		// Get Gateway name from actaboards-platform-gateway stack
		gatewayStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-platform-gateway/dev", nil)
		if err != nil {
			return err
		}

		GatewayName := gatewayStack.GetStringOutput(pulumi.String("GatewayName"))
		GatewayNamespace := gatewayStack.GetStringOutput(pulumi.String("GatewayNamespace"))

		// Get namespace from actaboards-api stack
		apiStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api/dev", nil)
		if err != nil {
			return err
		}

		NamespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		// Get ImagePullSecret from actaboards-api-image-pull-secret stack
		imagePullSecretStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api-image-pull-secret/dev", nil)
		if err != nil {
			return err
		}

		ImagePullSecretName := imagePullSecretStack.GetStringOutput(pulumi.String("ImagePullSecretName"))

		// Actaboards Web Deployment
		Deployment, err := appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-web"),
				Namespace: NamespaceName,
				Labels: pulumi.StringMap{
					"app": pulumi.String("actaboards-web"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("actaboards-web"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("actaboards-web"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ImagePullSecrets: corev1.LocalObjectReferenceArray{
							&corev1.LocalObjectReferenceArgs{
								Name: ImagePullSecretName,
							},
						},
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:            pulumi.String("actaboards-web"),
								Image:           pulumi.String("ghcr.io/actaboards/actaboards-web:main"),
								ImagePullPolicy: pulumi.String("Always"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
										Name:          pulumi.String("http"),
									},
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{
										"memory": pulumi.String("128Mi"),
										"cpu":    pulumi.String("100m"),
									},
									Limits: pulumi.StringMap{
										"memory": pulumi.String("512Mi"),
										"cpu":    pulumi.String("500m"),
									},
								},
							},
						},
					},
				},
			},
		})

		if err != nil {
			return err
		}

		// Actaboards Web Service
		Service, err := corev1.NewService(ctx, ns.Get("service"), &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-web"),
				Namespace: NamespaceName,
				Labels: pulumi.StringMap{
					"app": pulumi.String("actaboards-web"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app": pulumi.String("actaboards-web"),
				},
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(80),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Type: pulumi.String("ClusterIP"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{Deployment}))

		if err != nil {
			return err
		}

		// HTTPRoute for Web (acta.network)
		_, err = apiextensions.NewCustomResource(ctx, ns.Get("web-httproute"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-web-httproute"),
				Namespace: NamespaceName,
				Annotations: pulumi.StringMap{
					"external-dns.alpha.kubernetes.io/hostname": pulumi.String("acta.network"),
				},
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"parentRefs": pulumi.Array{
						pulumi.Map{
							"name":        GatewayName,
							"namespace":   GatewayNamespace,
							"kind":        pulumi.String("Gateway"),
							"sectionName": pulumi.String("https-acta"),
						},
					},
					"hostnames": pulumi.Array{
						pulumi.String("acta.network"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"matches": pulumi.Array{
								pulumi.Map{
									"path": pulumi.Map{
										"type":  pulumi.String("PathPrefix"),
										"value": pulumi.String("/"),
									},
								},
							},
							"backendRefs": pulumi.Array{
								pulumi.Map{
									"name": Service.Metadata.Name(),
									"port": pulumi.Int(80),
								},
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{Service}))

		if err != nil {
			return err
		}

		// HTTPRoute for HTTP to HTTPS redirect
		_, err = apiextensions.NewCustomResource(ctx, ns.Get("web-httproute-redirect"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-web-httproute-redirect"),
				Namespace: NamespaceName,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"parentRefs": pulumi.Array{
						pulumi.Map{
							"name":        GatewayName,
							"namespace":   GatewayNamespace,
							"kind":        pulumi.String("Gateway"),
							"sectionName": pulumi.String("http"),
						},
					},
					"hostnames": pulumi.Array{
						pulumi.String("acta.network"),
					},
					"rules": pulumi.Array{
						pulumi.Map{
							"filters": pulumi.Array{
								pulumi.Map{
									"type": pulumi.String("RequestRedirect"),
									"requestRedirect": pulumi.Map{
										"scheme":     pulumi.String("https"),
										"statusCode": pulumi.Int(301),
									},
								},
							},
						},
					},
				},
			},
		})

		if err != nil {
			return err
		}

		ctx.Export("hostname", pulumi.String("https://acta.network"))

		return nil
	})
}
