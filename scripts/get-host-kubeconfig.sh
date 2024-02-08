#!/usr/bin/env bash

parse_parameters(){
  while [[ -n "$1" ]]; do
    case $1 in
      --server) shift
                server=$1
                ;;
      *)        echo "Unknown arg $1"
                exit 1
    esac
    shift
  done

  if [[ -z $server ]]; then
    echo "Server parameter missed but required"
    exit 1
  fi
}

echo_kubeconfig_base64(){
  cert=$(kubectl get secret virtualization-csi-driver-secret -ojson | jq -r '.data."ca.crt"')
  token=$(kubectl get secret virtualization-csi-driver-secret -ojson | jq -r '.data.token' | base64 --decode)
  namespace=$(kubectl get secret virtualization-csi-driver-secret -ojson | jq -r '.data.namespace' | base64 --decode)

  config="""apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: $cert
    server: $server
  name: csi
contexts:
- context:
    cluster: csi
    namespace: $namespace
    user: csi
  name: csi@csi
current-context: csi@csi
kind: Config
preferences: {}
users:
- name: csi
  user:
    token: $token"""
  echo "$config" | base64 -w 0
  echo
}

main(){
  parse_parameters $@
  echo_kubeconfig_base64
}

main $@
