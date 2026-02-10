package main

import (
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		apiStack, err := pulumi.NewStackReference(ctx, "mirrorboards/core-xauth/dev", nil)
		if err != nil {
			return err
		}

		NamespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		_, err = apiextensions.NewCustomResource(ctx, "dragonfly", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("dragonflydb.io/v1alpha1"),
			Kind:       pulumi.String("Dragonfly"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("dragonfly"),
				Namespace: NamespaceName,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/name":       pulumi.String("dragonfly"),
					"app.kubernetes.io/instance":   pulumi.String("dragonfly-sample"),
					"app.kubernetes.io/part-of":    pulumi.String("dragonfly-operator"),
					"app.kubernetes.io/managed-by": pulumi.String("kustomize"),
					"app.kubernetes.io/created-by": pulumi.String("dragonfly-operator"),
				},
			},
			OtherFields: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 1,
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "500m",
							"memory": "500Mi",
						},
						"limits": map[string]interface{}{
							"cpu":    "600m",
							"memory": "750Mi",
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("DragonflyServiceName", pulumi.Sprintf("dragonfly.%s.svc.cluster.local", NamespaceName))

		return nil
	})
}
