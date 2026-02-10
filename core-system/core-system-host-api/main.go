package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get namespace from core-system stack
		coreSystemStack, err := pulumi.NewStackReference(ctx, "mirrorboards/core-system/dev", nil)
		if err != nil {
			return err
		}
		NamespaceName := coreSystemStack.GetStringOutput(pulumi.String("NamespaceName"))

		// Get Postgres secret name from core-system-db-postgres stack
		postgresStack, err := pulumi.NewStackReference(ctx, "mirrorboards/core-system-db-postgres/dev", nil)
		if err != nil {
			return err
		}
		PostgresSecretName := postgresStack.GetStringOutput(pulumi.String("PostgresSecretName"))

		appLabels := pulumi.StringMap{
			"app": pulumi.String("systemboards-api"),
		}

		Deployment, err := appsv1.NewDeployment(ctx, "core-system-host-api-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("systemboards-api"),
				Namespace: NamespaceName,
				Labels:    appLabels,
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
								Name:            pulumi.String("systemboards-api"),
								Image:           pulumi.String("ghcr.io/systemboards/systemboards-api:main"),
								ImagePullPolicy: pulumi.String("Always"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(3003),
										Name:          pulumi.String("http"),
									},
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("PORT"),
										Value: pulumi.String("3003"),
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("SYSTEM_POSTGRES_URI"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("uri"),
											},
										},
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

		Service, err := corev1.NewService(ctx, "core-system-host-api-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("systemboards-api"),
				Namespace: NamespaceName,
				Labels:    appLabels,
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: appLabels,
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(3003),
						TargetPort: pulumi.Int(3003),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Type: pulumi.String("ClusterIP"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{Deployment}))
		if err != nil {
			return err
		}

		ctx.Export("DeploymentName", Deployment.Metadata.Name())
		ctx.Export("ServiceName", Service.Metadata.Name())

		return nil
	})
}
