package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("mirrorboards", "platform")

		_, err := apiextensions.NewCustomResource(ctx, ns.Get("pulumi-stacks-mirrorboard-xxx"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("pulumi.com/v1"),
			Kind:       pulumi.String("Stack"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("mirrorboard-xxx"),
				Namespace: pulumi.String("pulumi-stacks"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"serviceAccountName": pulumi.String("pulumi"),
					"stack":              pulumi.String("mirrorboards/mirrorboard/xxx"),
					"fluxSource": pulumi.Map{
						"sourceRef": pulumi.Map{
							"apiVersion": pulumi.String("source.toolkit.fluxcd.io/v1"),
							"kind":       pulumi.String("GitRepository"),
							"name":       pulumi.String("mirrorboards-stacks"),
						},
						"dir": pulumi.String("mirrorboard/mirrorboard"),
					},
					"envRefs": pulumi.Map{
						"PULUMI_ACCESS_TOKEN": pulumi.Map{
							"type": pulumi.String("Secret"),
							"secret": pulumi.Map{
								"name": pulumi.String("pulumi-api-secret"),
								"key":  pulumi.String("accessToken"),
							},
						},
						"PULUMI_CONFIG_PASSPHRASE": pulumi.Map{
							"type": pulumi.String("Secret"),
							"secret": pulumi.Map{
								"name": pulumi.String("pulumi-config-passphrase"),
								"key":  pulumi.String("passphrase"),
							},
						},
						"GOWORK": pulumi.Map{
							"type": pulumi.String("Literal"),
							"literal": pulumi.Map{
								"value": pulumi.String("off"),
							},
						},
					},
					"destroyOnFinalize": pulumi.Bool(true),
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(ctx, ns.Get("pulumi-stacks-mirrorboard-yyy"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("pulumi.com/v1"),
			Kind:       pulumi.String("Stack"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("mirrorboard-yyy"),
				Namespace: pulumi.String("pulumi-stacks"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"serviceAccountName": pulumi.String("pulumi"),
					"stack":              pulumi.String("mirrorboards/mirrorboard/yyy"),
					"fluxSource": pulumi.Map{
						"sourceRef": pulumi.Map{
							"apiVersion": pulumi.String("source.toolkit.fluxcd.io/v1"),
							"kind":       pulumi.String("GitRepository"),
							"name":       pulumi.String("mirrorboards-stacks"),
						},
						"dir": pulumi.String("mirrorboard/mirrorboard"),
					},
					"envRefs": pulumi.Map{
						"PULUMI_ACCESS_TOKEN": pulumi.Map{
							"type": pulumi.String("Secret"),
							"secret": pulumi.Map{
								"name": pulumi.String("pulumi-api-secret"),
								"key":  pulumi.String("accessToken"),
							},
						},
						"PULUMI_CONFIG_PASSPHRASE": pulumi.Map{
							"type": pulumi.String("Secret"),
							"secret": pulumi.Map{
								"name": pulumi.String("pulumi-config-passphrase"),
								"key":  pulumi.String("passphrase"),
							},
						},
						"GOWORK": pulumi.Map{
							"type": pulumi.String("Literal"),
							"literal": pulumi.Map{
								"value": pulumi.String("off"),
							},
						},
					},
					"destroyOnFinalize": pulumi.Bool(true),
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
}
