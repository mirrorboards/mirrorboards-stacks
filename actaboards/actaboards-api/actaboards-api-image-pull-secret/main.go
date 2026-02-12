package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		githubCfg := config.New(ctx, "github")

		cluster, err := stacks.NewDigitalOceanClusterFromStack(ctx, ns.Get("cluster"), &stacks.DigitalOceanClusterFromStackArgs{
			StackReference: "organization/actaboards/dev",
		})
		if err != nil {
			return err
		}

		apiStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-api/dev", nil)
		if err != nil {
			return err
		}
		namespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		imagePullSecret, err := corev1.NewSecret(ctx, ns.Get("image-pull-secret"), &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("image-pull-secret")),
				Namespace: namespaceName,
			},
			Type:       pulumi.String("kubernetes.io/dockerconfigjson"),
			StringData: stacks.GenerateDockerPullImageConfigJSON("ghcr.io", pulumi.String(githubCfg.Get("username")), githubCfg.RequireSecret("token")),
		}, pulumi.Provider(cluster.Provider))
		if err != nil {
			return err
		}

		ctx.Export("ImagePullSecretName", imagePullSecret.Metadata.Name())
		ctx.Export("ImagePullSecretNamespace", imagePullSecret.Metadata.Namespace())

		return nil
	})
}
