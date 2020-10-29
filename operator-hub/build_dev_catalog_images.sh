OPM=./bin/opm

CURR_DIR=$(pwd)
OPERATOR_REGISTRY_REPO=~/go/src/github.com/operator-registry

# MANIFESTS_DIR the location of the nvmesh_bundle dir relative to the operator-registry repo root
MANIFESTS_DIR=./nvmesh_bundle/manifests

CONFIG_FILE=../manifests/config.yaml
OPER_VERSION=$(yq r $CONFIG_FILE "operator.version")
BUNDLE_RELEASE=$(yq r $CONFIG_FILE "bundle.release")
INDEX_BUILD=$(yq r $CONFIG_FILE "bundle.dev.index_build")
BUNDLE_IMG=$(yq r $CONFIG_FILE "bundle.dev.bundle_image_name")
INDEX_IMG=$(yq r $CONFIG_FILE "bundle.dev.index_image_name")
BUNDLE_VERSION="${OPER_VERSION}-${BUNDLE_RELEASE}-${INDEX_BUILD}"
BUNDLE_IMAGE_NAME="${BUNDLE_IMG}:${BUNDLE_VERSION}"
INDEX_IMAGE_NAME="${INDEX_IMG}:${BUNDLE_VERSION}"
PACKAGE_NAME=nvmesh-operator
CHANNEL=beta

echo "BUNDLE_IMAGE_NAME=$BUNDLE_IMAGE_NAME"
echo "INDEX_IMAGE_NAME=$INDEX_IMAGE_NAME"

exit_if_err() {
    rc=$1
    msg=$2
    if [ $rc -ne 0 ]; then
        echo $msg
        exit $rc
    fi
}

copy_sources_to_operator_registry_repo() {
    BUNDLE_SOURCE_DIR=./catalog_bundle/
    BUNDLE_TARGET_DIR=$OPERATOR_REGISTRY_REPO/nvmesh_bundle

    if [ ! -d "$BUNDLE_TARGET_DIR" ]; then
        mkdir -p "$BUNDLE_TARGET_DIR"
    else
        sudo rm -rf $BUNDLE_TARGET_DIR/*
    fi

    echo "Copying bundle data from $BUNDLE_SOURCE_DIR to $BUNDLE_TARGET_DIR"
    cp -r "$BUNDLE_SOURCE_DIR"/* "$BUNDLE_TARGET_DIR"
    return $?
}

copy_sources_to_operator_registry_repo
exit_if_err $? "Failed to copy bundle sources to the operator-registry repo"

echo "changing dir to  $OPERATOR_REGISTRY_REPO"
cd $OPERATOR_REGISTRY_REPO

echo "Generation Dokcerfile for bundle image"
$OPM alpha bundle generate --directory $MANIFESTS_DIR --package $PACKAGE_NAME --channels $CHANNEL --default $CHANNEL
exit_if_err $? "Failed to generate dockerfile for bundle image"

echo "Building Bundle Image"
$OPM alpha bundle build --directory $MANIFESTS_DIR --tag $BUNDLE_IMAGE_NAME --package $PACKAGE_NAME --channels $CHANNEL --default $CHANNEL
exit_if_err $? "Failed to build bundle image. cmd=$OPM alpha bundle build --directory $MANIFESTS_DIR --tag $BUNDLE_IMAGE_NAME --package $PACKAGE_NAME --channels $CHANNEL --default $CHANNEL"

echo "Upload bundle image to image registry..."
docker push $BUNDLE_IMAGE_NAME

echo "Verifing Bundle Image"
$OPM alpha bundle validate --tag $BUNDLE_IMAGE_NAME --image-builder docker #--skip-tls
exit_if_err $? "Failed to verify Bundle Image"

echo "Adding to index"
$OPM index add --bundles $BUNDLE_IMAGE_NAME --tag $INDEX_IMAGE_NAME #--skip-tls
exit_if_err $? "Failed to run $OPM index add --bundles $BUNDLE_IMAGE_NAME --tag $INDEX_IMAGE_NAME"

podman push $INDEX_IMAGE_NAME #--tls-verify=false
exit_if_err $? "Failed to push index image. command: podman push $INDEX_IMAGE_NAME"

echo "Editing dev/catalog_source.yaml"
cd $CURR_DIR
echo "yq w -i dev/catalog_source.yaml 'spec.image' $INDEX_IMAGE_NAME"
yq w -i dev/catalog_source.yaml 'spec.image' $INDEX_IMAGE_NAME

echo "Done."
