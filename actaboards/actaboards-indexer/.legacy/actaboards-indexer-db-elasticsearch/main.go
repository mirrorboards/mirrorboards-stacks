package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "indexer")

		cluster, err := stacks.NewDigitalOceanClusterFromStack(ctx, ns.Get("cluster"), &stacks.DigitalOceanClusterFromStackArgs{
			StackReference: "organization/actaboards/dev",
		})
		if err != nil {
			return err
		}

		// Get namespace from actaboards-indexer stack
		indexerStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-indexer/dev", nil)
		if err != nil {
			return err
		}

		NamespaceName := indexerStack.GetStringOutput(pulumi.String("NamespaceName"))

		// Elasticsearch 7.10.1 - compatible with actaboards-core
		ElasticsearchCluster, err := apiextensions.NewCustomResource(ctx, ns.Get("elasticsearch"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("elasticsearch.k8s.elastic.co/v1"),
			Kind:       pulumi.String("Elasticsearch"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("elasticsearch")),
				Namespace: NamespaceName,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"version": pulumi.String("7.10.1"),
					// Disable TLS for internal cluster communication (actaboards-core doesn't handle self-signed certs well)
					"http": pulumi.Map{
						"tls": pulumi.Map{
							"selfSignedCertificate": pulumi.Map{
								"disabled": pulumi.Bool(true),
							},
						},
					},
					"nodeSets": pulumi.Array{
						pulumi.Map{
							"name":  pulumi.String("default"),
							"count": pulumi.Int(1),
							"config": pulumi.Map{
								"node.store.allow_mmap": pulumi.Bool(false),
							},
							"volumeClaimTemplates": pulumi.Array{
								pulumi.Map{
									"metadata": pulumi.Map{
										"name": pulumi.String("elasticsearch-data"),
									},
									"spec": pulumi.Map{
										"accessModes": pulumi.StringArray{
											pulumi.String("ReadWriteOnce"),
										},
										"resources": pulumi.Map{
											"requests": pulumi.Map{
												"storage": pulumi.String("10Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(cluster.Provider))

		if err != nil {
			return err
		}

		// ECK creates a secret named <cluster-name>-es-http-certs-public for TLS
		// and <cluster-name>-es-elastic-user for credentials
		ElasticsearchSecretName := pulumi.String(ns.Get("elasticsearch") + "-es-elastic-user")
		ElasticsearchServiceName := pulumi.String(ns.Get("elasticsearch") + "-es-http")

		// Kibana 7.10.1 - compatible with Elasticsearch
		// Keep TLS enabled for Kibana (browser access needs secure cookies)
		Kibana, err := apiextensions.NewCustomResource(ctx, ns.Get("kibana"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("kibana.k8s.elastic.co/v1"),
			Kind:       pulumi.String("Kibana"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("kibana")),
				Namespace: NamespaceName,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"version": pulumi.String("7.10.1"),
					"count":   pulumi.Int(1),
					"elasticsearchRef": pulumi.Map{
						"name": pulumi.String(ns.Get("elasticsearch")),
					},
				},
			},
		}, pulumi.Provider(cluster.Provider), pulumi.DependsOn([]pulumi.Resource{ElasticsearchCluster}))

		if err != nil {
			return err
		}

		KibanaServiceName := pulumi.String(ns.Get("kibana") + "-kb-http")

		ctx.Export("ElasticsearchClusterName", ElasticsearchCluster.Metadata.Name())
		ctx.Export("ElasticsearchSecretName", ElasticsearchSecretName)
		ctx.Export("ElasticsearchServiceName", ElasticsearchServiceName)
		ctx.Export("KibanaName", Kibana.Metadata.Name())
		ctx.Export("KibanaServiceName", KibanaServiceName)

		return nil
	})
}
