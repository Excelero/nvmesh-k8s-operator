FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=nvmesh-operator
LABEL operators.operatorframework.io.bundle.channels.v1=beta
LABEL operators.operatorframework.io.bundle.channel.default.v1=beta
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.0.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v2
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1

# OpenShift Labels
# Documentation: https://redhat-connect.gitbook.io/certified-operator-guide/ocp-deployment/operator-metadata/bundle-directory
# This lists openshift versions, starting with 4.5, that your operator will support. You have to start the version with a 'v', and no spaces are allowed
LABEL com.redhat.openshift.versions="v4.5-v4.9"
LABEL com.redhat.delivery.operator.bundle=true

# backport flag is used to indicate support for OpenShift versions before 4.5.
LABEL com.redhat.delivery.backport=true

COPY operator-hub/catalog_bundle/manifests /manifests/
COPY operator-hub/catalog_bundle/metadata /metadata/
