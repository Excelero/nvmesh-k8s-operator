# Integration with Operator Hub
This directory contains sources, scripts and documentation on how to buildand deploy the operator into OpenShift Operator-Hub

## Adding the Operator to OpenShift Operator-Hub listing:
### **CatalogSource**
   1. We will add a 'CatalogSource' Resource to an OpenShift cluster
   2. The CatalogSource yaml is located in ./catalog_source/catalog_source.yaml
   3. This yaml let's OpenShift know of our own list of operators holding our info about each operator
   4. CatalogSource can be viewed in the WebUI `Administration > Cluster Settings > Global Configuration > OperatorHub > Sources`
### **index-image** and the **Operator-Registry Server**
   1. Once a CatalogSource is added - OpenShift will start a pod with the 'index-image' (excelero/dev-os-catalog-source-index:0.0.0-[x]]) to serve the catalog index
   2. The pod will be created in the namespace "OpenShift-marketplace" with a name similar to *nvmesh-catalog-sb7jw*
   3. The index-image runs the operator-registry server which serves the catalog index
   4. The catalog index that this pod is serving is a list of available operators and the info of how to fetch each one of them
   5. The info in the index is just basic info that tells OpenShift how to display each of it's Operators in the Market-Place view
   6. An operator-registry server can have multiple Operators listed in the index - we currently use only one.
   7. Once the Operator-Registry pod is running successfully we should see our operator listed in the `Operators > OperatorHub` view
### **bundle-image**
   1. Once a specific Operator is selected for installation - The operator-registry server fetches another docker image called the 'bundle-image'
   2. The bundle-image is a docker image that stores the manifest required to deploy the operator and the nvmesh crd in the kubernetes cluster, it's not runnable
   3. The bundle image has the 2 directories manifests and metadata that appear under this repo in operator-hub/catalog_bundle

## Building the images
### How to build the images
   1. The tools used to build the images are located in the operator-framework/operator-registry (https://github.com/operator-framework/operator-registry)
   2. In order to build the images we need to clone the operator-registry repo, recommended location is: /home/{user}/go/src/github.com/operator-registry
   3. To build both the bundle-image and the index-image open the ./build_catalog_images.sh file for editing (located in this folder) and update the following:
      1. Update the OPERATOR_REGISTRY_REPO parameter to where you cloned the operator-registry repo
      2. Increment the VERSION parameter (otherwise the catalog pod will not fetch the updated image)
   4. Run the `./build_catalog_images.sh` script - the script will copy the bundle folder and build both index-image and bundle-image
   5. Update the index image with the new version in the image field in `catalog_source/catalog_source.yaml` file
   6. To update the catalog pod Run `kubectl apply -f operator-hub/catalog_source/catalog_source.yaml`
   7. The `./build_catalog_images.sh` script pushes the images to DockerHub under the Excelero repo - login into DockerHub with the excelero credentials is required
### Changes that require images rebuild
The following changes require rebuilding the index-image and bundle-image and updating the catalog source to point to the new index image:
   1. Changes to the CRD
   2. Changes to nvmesh_types.go (this changes the CRD by running make manifests)
   3. ClusterServiceVersion Changes


## For more details on serving operators please go to https://github.com/operator-framework/operator-registry