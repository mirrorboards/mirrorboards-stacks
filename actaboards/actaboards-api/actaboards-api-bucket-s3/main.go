package main

import (
	"fmt"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ns := namespace.NewNamespace("actaboards", "api")

		awsCfg := config.New(ctx, "aws")
		s3Cfg := config.New(ctx, "s3")

		awsRegion := awsCfg.Require("region")
		bucketName := s3Cfg.Require("bucketName")

		endpointUrl := fmt.Sprintf("https://s3.%s.amazonaws.com", awsRegion)
		publicUrlPrefix := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, awsRegion)

		// Get namespace from actaboards-api stack
		apiStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-api/dev", nil)
		if err != nil {
			return err
		}
		namespaceName := apiStack.GetStringOutput(pulumi.String("NamespaceName"))

		// --- S3 Bucket resources ---

		bucket, err := apiextensions.NewCustomResource(ctx, ns.Get("s3", "bucket"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("s3.aws.upbound.io/v1beta2"),
			Kind:       pulumi.String("Bucket"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"region": pulumi.String(awsRegion),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		publicAccessBlock, err := apiextensions.NewCustomResource(ctx, ns.Get("s3", "public-access-block"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("s3.aws.upbound.io/v1beta2"),
			Kind:       pulumi.String("BucketPublicAccessBlock"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-public-access"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"region":                pulumi.String(awsRegion),
						"blockPublicAcls":       pulumi.Bool(false),
						"blockPublicPolicy":     pulumi.Bool(false),
						"ignorePublicAcls":      pulumi.Bool(false),
						"restrictPublicBuckets": pulumi.Bool(false),
						"bucketRef": pulumi.Map{
							"name": pulumi.String(bucketName),
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{bucket}))
		if err != nil {
			return err
		}

		ownershipControls, err := apiextensions.NewCustomResource(ctx, ns.Get("s3", "ownership-controls"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("s3.aws.upbound.io/v1beta2"),
			Kind:       pulumi.String("BucketOwnershipControls"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-ownership"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"region": pulumi.String(awsRegion),
						"bucketRef": pulumi.Map{
							"name": pulumi.String(bucketName),
						},
						"rule": pulumi.MapArray{
							pulumi.Map{
								"objectOwnership": pulumi.String("BucketOwnerPreferred"),
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{bucket}))
		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(ctx, ns.Get("s3", "cors"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("s3.aws.upbound.io/v1beta2"),
			Kind:       pulumi.String("BucketCorsConfiguration"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-cors"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"region": pulumi.String(awsRegion),
						"bucketRef": pulumi.Map{
							"name": pulumi.String(bucketName),
						},
						"corsRule": pulumi.MapArray{
							pulumi.Map{
								"allowedOrigins": pulumi.ToStringArray([]string{"*"}),
								"allowedMethods": pulumi.ToStringArray([]string{"GET", "PUT", "POST", "DELETE", "HEAD"}),
								"allowedHeaders": pulumi.ToStringArray([]string{"*"}),
								"maxAgeSeconds":  pulumi.Int(3600),
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{bucket, publicAccessBlock, ownershipControls}))
		if err != nil {
			return err
		}

		// --- IAM User with scoped S3 permissions ---

		iamUserName := bucketName + "-s3-user"
		crossplaneSecretName := bucketName + "-s3-creds"

		iamUser, err := apiextensions.NewCustomResource(ctx, ns.Get("iam", "user"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("iam.aws.upbound.io/v1beta1"),
			Kind:       pulumi.String("User"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(iamUserName),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{bucket}))
		if err != nil {
			return err
		}

		policyDocument := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:PutObjectAcl",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:GetBucketLocation"
      ],
      "Resource": [
        "arn:aws:s3:::%s",
        "arn:aws:s3:::%s/*"
      ]
    }
  ]
}`, bucketName, bucketName)

		iamPolicy, err := apiextensions.NewCustomResource(ctx, ns.Get("iam", "policy"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("iam.aws.upbound.io/v1beta1"),
			Kind:       pulumi.String("Policy"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-s3-policy"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"policy": pulumi.String(policyDocument),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(ctx, ns.Get("iam", "policy-attachment"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("iam.aws.upbound.io/v1beta1"),
			Kind:       pulumi.String("UserPolicyAttachment"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-s3-policy-attachment"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"policyArnRef": pulumi.Map{
							"name": pulumi.String(bucketName + "-s3-policy"),
						},
						"userRef": pulumi.Map{
							"name": pulumi.String(iamUserName),
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{iamUser, iamPolicy}))
		if err != nil {
			return err
		}

		// AccessKey — Crossplane writes credentials to Secret in external-secrets-store namespace
		_, err = apiextensions.NewCustomResource(ctx, ns.Get("iam", "access-key"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("iam.aws.upbound.io/v1beta1"),
			Kind:       pulumi.String("AccessKey"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(bucketName + "-s3-access-key"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"forProvider": pulumi.Map{
						"userRef": pulumi.Map{
							"name": pulumi.String(iamUserName),
						},
					},
					"writeConnectionSecretToRef": pulumi.Map{
						"name":      pulumi.String(crossplaneSecretName),
						"namespace": pulumi.String("external-secrets-store"),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{iamUser}))
		if err != nil {
			return err
		}

		// --- ExternalSecret: merge Crossplane credentials + static values into one Secret ---
		// Crossplane AccessKey writes keys: "attribute.id" (access key ID), "attribute.secret" (secret key)
		// Verify with: kubectl get secret <crossplaneSecretName> -n external-secrets-store -o jsonpath='{.data}' | jq

		finalSecretName := ns.Get("bucket", "s3")

		externalSecret, err := apiextensions.NewCustomResource(ctx, ns.Get("external-secret"), &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("external-secrets.io/v1"),
			Kind:       pulumi.String("ExternalSecret"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(finalSecretName),
				Namespace: namespaceName,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"refreshInterval": pulumi.String("1h"),
					"secretStoreRef": pulumi.Map{
						"name": pulumi.String("kubernetes-secret-store"),
						"kind": pulumi.String("ClusterSecretStore"),
					},
					"target": pulumi.Map{
						"name": pulumi.String(finalSecretName),
						"template": pulumi.Map{
							"engineVersion": pulumi.String("v2"),
							"data": pulumi.Map{
								"access_key_id":     pulumi.String("{{ .access_key_id }}"),
								"secret_access_key": pulumi.String("{{ .secret_access_key }}"),
								"endpoint_url":      pulumi.String(endpointUrl),
								"region":            pulumi.String(awsRegion),
								"bucket":            pulumi.String(bucketName),
								"public_url_prefix": pulumi.String(publicUrlPrefix),
							},
						},
					},
					"data": pulumi.MapArray{
						pulumi.Map{
							"secretKey": pulumi.String("access_key_id"),
							"remoteRef": pulumi.Map{
								"key":      pulumi.String(crossplaneSecretName),
								"property": pulumi.String("attribute.id"),
							},
						},
						pulumi.Map{
							"secretKey": pulumi.String("secret_access_key"),
							"remoteRef": pulumi.Map{
								"key":      pulumi.String(crossplaneSecretName),
								"property": pulumi.String("attribute.secret"),
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Export outputs
		ctx.Export("S3SecretName", pulumi.String(finalSecretName))
		ctx.Export("BucketName", bucket.Metadata.Name())
		ctx.Export("BucketRegion", pulumi.String(awsRegion))
		ctx.Export("BucketEndpoint", pulumi.String(endpointUrl))
		ctx.Export("BucketPublicUrlPrefix", pulumi.String(publicUrlPrefix))
		ctx.Export("ExternalSecretName", externalSecret.Metadata.Name())

		return nil
	})
}
