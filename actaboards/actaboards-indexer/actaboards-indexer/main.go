package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
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

		Namespace, err := corev1.NewNamespace(ctx, ns.Get("namespace"), &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(ns.Get()),
				Labels: pulumi.StringMap{
					"istio.io/dataplane-mode": pulumi.String("ambient"),
				},
			},
		}, pulumi.Provider(cluster.Provider))
		if err != nil {
			return err
		}

		ctx.Export("NamespaceName", Namespace.Metadata.Name())

		return nil
	})
}
