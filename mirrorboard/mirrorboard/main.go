package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("mirrorboard", ctx.Stack())

		Namespace, err := corev1.NewNamespace(ctx, ns.Get("namespace"), &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(ns.Get()),
			},
		})

		if err != nil {
			return err
		}

		ctx.Export("NamespaceName", Namespace.Metadata.Name())

		return nil
	})
}
