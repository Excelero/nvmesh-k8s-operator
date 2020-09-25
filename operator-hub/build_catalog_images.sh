OPM=./bin/opm

OPERATOR_REGISTRY_REPO=~/go/src/github.com/operator-registry

# MANIFESTS_DIR the location of the nvmesh_bundle dir relative to the operator-registry repo root
MANIFESTS_DIR=./nvmesh_bundle/manifests

VERSION=0.0.0-18
BUNDLE_IMAGE_NAME=docker.io/excelero/dev-os-bundle:$VERSION
INDEX_IMAGE_NAME=docker.io/excelero/dev-os-catalog-source-index:$VERSION
PACKAGE_NAME=nvmesh-operator

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
$OPM alpha bundle generate --directory $MANIFESTS_DIR --package $PACKAGE_NAME --channels alpha --default alpha
exit_if_err $? "Failed to generate dockerfile for bundle image"

echo "Building Bundle Image"
$OPM alpha bundle build --directory $MANIFESTS_DIR --tag $BUNDLE_IMAGE_NAME --package $PACKAGE_NAME --channels alpha --default alpha
exit_if_err $? "Failed to build bundle image. cmd=$OPM alpha bundle build --directory $MANIFESTS_DIR --tag $BUNDLE_IMAGE_NAME --package $PACKAGE_NAME --channels alpha --default alpha"

echo "Upload bundle image to image registry..."
docker push $BUNDLE_IMAGE_NAME

echo "Verifing Bundle Image"
$OPM alpha bundle validate --tag $BUNDLE_IMAGE_NAME --image-builder docker #--skip-tls
exit_if_err $? "Failed to verify Bundle Image"

echo "Adding to index"
$OPM index add --bundles $BUNDLE_IMAGE_NAME --tag $INDEX_IMAGE_NAME #--skip-tls
exit_if_err $? "Failed to run $OPM index add --bundles $BUNDLE_IMAGE_NAME --tag $INDEX_IMAGE_NAME"

podman push $INDEX_IMAGE_NAME #--tls-verify=false
exit_if_err $? "Failed to push index image. command: podman push $INDEX_IMAGE_NAME --tls-verify=false"


echo "Done."