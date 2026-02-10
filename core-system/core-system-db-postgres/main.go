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
		ns := namespace.NewNamespace("core-system")

		NamespaceName := stacks.GetStringStackOutput(ctx, "mirrorboards/core-system/dev", "NamespaceName")

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
		})
		if err != nil {
			return err
		}

		PostgresSecretName := pulumi.String(ns.Get("postgres") + "-app")

		ctx.Export("PostgresClusterName", PostgresCluster.Metadata.Name())
		ctx.Export("PostgresSecretName", PostgresSecretName)

		return nil
	})
}
