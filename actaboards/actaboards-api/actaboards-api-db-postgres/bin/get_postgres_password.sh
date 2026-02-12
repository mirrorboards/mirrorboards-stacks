#!/bin/bash

NAMESPACE="actaboards-api"
SECRET_NAME="actaboards-api-postgres-app"

echo "Password:"
kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.password}' | base64 -d
echo ""

echo ""
echo "Database URL:"
echo "$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.fqdn-uri}' | base64 -d)?sslmode=disable"
