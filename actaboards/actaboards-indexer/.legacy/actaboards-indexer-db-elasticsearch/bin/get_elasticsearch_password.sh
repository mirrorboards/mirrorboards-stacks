#!/bin/bash

NAMESPACE="actaboards-indexer"
SECRET_NAME="actaboards-indexer-elasticsearch-es-elastic-user"
SERVICE_NAME="actaboards-indexer-elasticsearch-es-http"

echo "User: elastic"
echo ""
echo "Password:"
kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.elastic}' | base64 -d
echo ""

echo ""
echo "Elasticsearch URL (internal, HTTP for actaboards-core):"
echo "http://$SERVICE_NAME.$NAMESPACE.svc.cluster.local:9200"

echo ""
echo "Kibana URL (internal, HTTPS):"
echo "https://actaboards-indexer-kibana-kb-http.$NAMESPACE.svc.cluster.local:5601"
