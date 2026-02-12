package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/charts"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		cluster, err := stacks.NewDigitalOceanClusterFromStack(ctx, ns.Get("cluster"), &stacks.DigitalOceanClusterFromStackArgs{
			StackReference: "organization/actaboards/dev",
		})
		if err != nil {
			return err
		}

		// Get namespace from actaboards-api stack
		apiStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-api/dev", nil)
		if err != nil {
			return err
		}

		NamespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		Dragonfly, err := charts.NewDragonflyInstance(ctx, ns.Get("dragonfly"), &charts.NewDragonflyInstanceArgs{
			Name:      pulumi.String("dragonfly"),
			Namespace: NamespaceName,
		}, pulumi.Provider(cluster.Provider))

		if err != nil {
			return err
		}

		ctx.Export("DragonflyServiceName", pulumi.Sprintf("dragonfly.%s.svc.cluster.local", NamespaceName))

		_ = Dragonfly

		return nil
	})
}
