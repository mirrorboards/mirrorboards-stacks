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

		PostgresCluster, err := apiextensions.NewCustomResource(ctx, ns.Get("postgres"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("postgresql.cnpg.io/v1"),
			Kind:       pulumi.String("Cluster"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("postgres")),
				Namespace: NamespaceName,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"instances": pulumi.Int(1),
					"storage": pulumi.Map{
						"size": pulumi.String("1Gi"),
					},
				},
			},
		}, pulumi.Provider(cluster.Provider))

		if err != nil {
			return err
		}

		PostgresSecretName := pulumi.String(ns.Get("postgres") + "-app")

		ctx.Export("PostgresClusterName", PostgresCluster.Metadata.Name())
		ctx.Export("PostgresSecretName", PostgresSecretName)

		return nil
	})
}
