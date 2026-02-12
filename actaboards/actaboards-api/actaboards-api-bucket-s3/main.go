package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		DigitalOceanCfg := config.New(ctx, "digitalocean")

		// Get K8s cluster for creating secrets
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

		SpacesProvider, err := digitalocean.NewProvider(ctx, ns.Get("provider", "spaces"), &digitalocean.ProviderArgs{
			Token:           DigitalOceanCfg.RequireSecret("token"),
			SpacesAccessId:  DigitalOceanCfg.RequireSecret("spaces_access_id"),
			SpacesSecretKey: DigitalOceanCfg.RequireSecret("spaces_secret_key"),
		})
		if err != nil {
			return err
		}

		region := pulumi.String("fra1")
		bucketName := pulumi.String("actaboards-network-storage-v2")
		endpointUrl := pulumi.Sprintf("https://%s.digitaloceanspaces.com", region)
		publicUrlPrefix := pulumi.Sprintf("https://%s.%s.digitaloceanspaces.com", bucketName, region)

		// Create S3-compatible Spaces bucket for file uploads
		UploadBucket, err := digitalocean.NewSpacesBucket(ctx, ns.Get("upload", "bucket"), &digitalocean.SpacesBucketArgs{
			Name:   bucketName,
			Acl:    pulumi.String("public-read"),
			Region: region,
			CorsRules: digitalocean.SpacesBucketCorsRuleArray{
				&digitalocean.SpacesBucketCorsRuleArgs{
					AllowedOrigins: pulumi.StringArray{
						pulumi.String("*"),
					},
					AllowedMethods: pulumi.StringArray{
						pulumi.String("GET"),
						pulumi.String("PUT"),
						pulumi.String("POST"),
						pulumi.String("DELETE"),
						pulumi.String("HEAD"),
					},
					AllowedHeaders: pulumi.StringArray{
						pulumi.String("*"),
					},
					MaxAgeSeconds: pulumi.Int(3600),
				},
			},
		}, pulumi.Provider(SpacesProvider), pulumi.Protect(false))
		if err != nil {
			return err
		}

		// Create K8s Secret with S3 credentials
		S3Secret, err := corev1.NewSecret(ctx, ns.Get("s3", "secret"), &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("bucket", "s3")),
				Namespace: NamespaceName,
			},
			Type: pulumi.String("Opaque"),
			StringData: pulumi.StringMap{
				"endpoint_url":      endpointUrl,
				"region":            region,
				"bucket":            bucketName,
				"public_url_prefix": publicUrlPrefix,
				"access_key_id":     DigitalOceanCfg.RequireSecret("spaces_access_id"),
				"secret_access_key": DigitalOceanCfg.RequireSecret("spaces_secret_key"),
			},
		}, pulumi.Provider(cluster.Provider))
		if err != nil {
			return err
		}

		// Export outputs
		ctx.Export("S3SecretName", S3Secret.Metadata.Name())
		ctx.Export("BucketName", UploadBucket.Name)
		ctx.Export("BucketRegion", region)
		ctx.Export("BucketEndpoint", endpointUrl)
		ctx.Export("BucketPublicUrlPrefix", publicUrlPrefix)

		return nil
	})
}
