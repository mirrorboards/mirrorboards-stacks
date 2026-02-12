package main

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/blockchain/actaboards"
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/mirrorboards-go/mirrorboards-pulumi/stacks"

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

		indexerStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-indexer/dev", nil)
		if err != nil {
			return err
		}

		namespaceName := indexerStack.GetStringOutput(pulumi.String("NamespaceName"))

		genesisStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-network-genesis/dev", nil)
		if err != nil {
			return err
		}

		genesisURL := genesisStack.GetStringOutput(pulumi.String("GenesisURL"))

		postgresStack, err := pulumi.NewStackReference(ctx, "organization/actaboards-indexer-db-postgres/dev", nil)
		if err != nil {
			return err
		}

		postgresSecretName := postgresStack.GetStringOutput(pulumi.String("PostgresSecretName"))

		_, err = actaboards.NewIndexer(ctx, ns.Get("node", "postgres-indexer"), &actaboards.IndexerArgs{
			Name:       pulumi.String(ns.Get("node", "postgres-indexer")),
			Namespace:  namespaceName,
			Image:      pulumi.String("ghcr.io/actaboards/actaboards-core:latest"),
			GenesisURL: genesisURL,
			SeedNodes: pulumi.StringArray{
				pulumi.String("node01.acta.network:2771"),
				pulumi.String("node02.acta.network:2771"),
			},
			Plugins: pulumi.StringArray{
				pulumi.String("witness"),
				pulumi.String("postgres_indexer"),
			},
			PostgresIndexerSecretName:      postgresSecretName,
			PostgresIndexerSecretKey:       pulumi.String("uri"),
			PostgresIndexerMode:            pulumi.Int(2),
			PostgresIndexerOperationString: pulumi.Bool(true),
			PostgresIndexerVisitor:         pulumi.Bool(true),
			PostgresIndexerStartBlock:      pulumi.Int(0),
		}, pulumi.Provider(cluster.Provider))
		if err != nil {
			return err
		}

		return nil
	})
}
