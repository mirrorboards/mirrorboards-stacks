package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/charts"
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		// Get namespace from actaboards-api stack
		apiStack, err := pulumi.NewStackReference(ctx, "mirrorboards/actaboards-api/dev", nil)
		if err != nil {
			return err
		}

		NamespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		Dragonfly, err := charts.NewDragonflyInstance(ctx, ns.Get("dragonfly"), &charts.NewDragonflyInstanceArgs{
			Name:      pulumi.String("dragonfly"),
			Namespace: NamespaceName,
		})

		if err != nil {
			return err
		}

		ctx.Export("DragonflyServiceName", pulumi.Sprintf("dragonfly.%s.svc.cluster.local", NamespaceName))

		_ = Dragonfly

		return nil
	})
}
