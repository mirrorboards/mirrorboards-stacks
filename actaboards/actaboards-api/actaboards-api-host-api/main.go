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

		// Get Postgres secret name from actaboards-api-db-postgres stack
		postgresStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api-db-postgres/dev", nil)
		if err != nil {
			return err
		}

		PostgresSecretName := postgresStack.GetStringOutput(pulumi.String("PostgresSecretName"))

		// Get Redis service name from actaboards-api-db-redis stack
		redisStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api-db-redis/dev", nil)
		if err != nil {
			return err
		}

		RedisServiceName := redisStack.GetStringOutput(pulumi.String("DragonflyServiceName"))

		// Get S3 secret name from actaboards-api-bucket-s3 stack
		s3Stack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api-bucket-s3/dev", nil)
		if err != nil {
			return err
		}

		S3SecretName := s3Stack.GetStringOutput(pulumi.String("S3SecretName"))

		// Get Indexer Postgres secret from actaboards-indexer-db-postgres stack
		indexerPostgresStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-indexer-db-postgres/dev", nil)
		if err != nil {
			return err
		}

		IndexerPostgresSecretName := indexerPostgresStack.GetStringOutput(pulumi.String("PostgresSecretName"))

		// Get Indexer namespace from actaboards-indexer stack
		indexerStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-indexer/dev", nil)
		if err != nil {
			return err
		}

		IndexerNamespaceName := indexerStack.GetStringOutput(pulumi.String("NamespaceName"))

		// Look up the indexer postgres secret and copy its URI to the API namespace
		indexerPostgresSecret, err := corev1.GetSecret(ctx, ns.Get("indexer-postgres-secret-lookup"),
			pulumi.All(IndexerNamespaceName, IndexerPostgresSecretName).ApplyT(func(args []any) pulumi.ID {
				return pulumi.ID(args[0].(string) + "/" + args[1].(string))
			}).(pulumi.IDOutput),
			nil,
		)
		if err != nil {
			return err
		}

		// Create a copy of the indexer postgres secret in the API namespace
		indexerPostgresSecretCopy, err := corev1.NewSecret(ctx, ns.Get("indexer-postgres-secret"), &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-indexer-postgres-app"),
				Namespace: NamespaceName,
			},
			Data: pulumi.StringMap{
				"uri": indexerPostgresSecret.Data.ApplyT(func(data map[string]string) string {
					return data["uri"]
				}).(pulumi.StringOutput),
			},
		})
		if err != nil {
			return err
		}

		IndexerPostgresSecretCopyName := indexerPostgresSecretCopy.Metadata.Name()

		// Actaboards API Deployment
		Deployment, err := appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-api"),
				Namespace: NamespaceName,
				Labels: pulumi.StringMap{
					"app": pulumi.String("actaboards-api"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("actaboards-api"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("actaboards-api"),
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
								Name:            pulumi.String("actaboards-api"),
								Image:           pulumi.String("ghcr.io/actaboards/actaboards-api:main"),
								ImagePullPolicy: pulumi.String("Always"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(3000),
										Name:          pulumi.String("http"),
									},
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ENVIRONMENT"),
										Value: pulumi.String("production"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("PORT"),
										Value: pulumi.String("3000"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("VAULT_REDIS_CONNECTION_URL"),
										Value: pulumi.Sprintf("redis://%s", RedisServiceName),
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_URI"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("uri"),
											},
										},
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_HOST"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("host"),
											},
										},
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_PORT"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("port"),
											},
										},
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_DB"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("dbname"),
											},
										},
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_USER"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("username"),
											},
										},
									},
									&corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_PASSWORD"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: PostgresSecretName,
												Key:  pulumi.String("password"),
											},
										},
									},
								// Indexer Postgres URI (cross-namespace secret copy)
								&corev1.EnvVarArgs{
									Name: pulumi.String("INDEXER_POSTGRES_URI"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: IndexerPostgresSecretCopyName,
											Key:  pulumi.String("uri"),
										},
									},
								},
								// S3 configuration
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_ENDPOINT_URL"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("endpoint_url"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_REGION"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("region"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_BUCKET"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("bucket"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_PUBLIC_URL_PREFIX"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("public_url_prefix"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_ACCESS_KEY_ID"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("access_key_id"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("S3_SECRET_ACCESS_KEY"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: S3SecretName,
											Key:  pulumi.String("secret_access_key"),
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

		// Actaboards API Service
		Service, err := corev1.NewService(ctx, ns.Get("service"), &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-api"),
				Namespace: NamespaceName,
				Labels: pulumi.StringMap{
					"app": pulumi.String("actaboards-api"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app": pulumi.String("actaboards-api"),
				},
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(3000),
						TargetPort: pulumi.Int(3000),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Type: pulumi.String("ClusterIP"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{Deployment}))

		if err != nil {
			return err
		}

		// HTTPRoute for API (api.acta.network)
		_, err = apiextensions.NewCustomResource(ctx, ns.Get("api-httproute"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-api-httproute"),
				Namespace: NamespaceName,
				Annotations: pulumi.StringMap{
					"external-dns.alpha.kubernetes.io/hostname": pulumi.String("api.acta.network"),
				},
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"parentRefs": pulumi.Array{
						pulumi.Map{
							"name":        GatewayName,
							"namespace":   GatewayNamespace,
							"kind":        pulumi.String("Gateway"),
							"sectionName": pulumi.String("https-api-acta"),
						},
					},
					"hostnames": pulumi.Array{
						pulumi.String("api.acta.network"),
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
									"port": pulumi.Int(3000),
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
		_, err = apiextensions.NewCustomResource(ctx, ns.Get("api-httproute-redirect"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
			Kind:       pulumi.String("HTTPRoute"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("actaboards-api-httproute-redirect"),
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
						pulumi.String("api.acta.network"),
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

		ctx.Export("hostname-api", pulumi.String("https://api.acta.network"))

		return nil
	})
}
