#!/usr/bin/env python2

import json
import yaml
import sys
from os import path
from shutil import copyfile, copytree, rmtree

output_file = "../operator-hub/catalog_bundle/manifests/cluster_service_version.yaml"
config_file = "./config.yaml"

bases = "./bases"
deploy = "../deploy"
bundle = "../operator-hub/catalog_bundle"

crd_base = path.join(bases, "crd/nvmesh.crd.yaml")
csv_base = path.join(bases, "csv/csv.yaml")
role_file = path.join(bases, "rbac/role.yaml")
operator_dep_file = path.join(bases, "operator/deployment.yaml")
service_account_file = path.join(bases, "extra/service_account.yaml")

def load_yaml_file(filename):
    with open(filename, 'r') as f:
        return yaml.safe_load(f)

def write_yaml_file(obj, output_file):
    with open(output_file, 'w') as f:
        yaml.dump(obj, f, sort_keys=True)

def get_name(obj):
    return obj['metadata']['name']

config = load_yaml_file(config_file)
version_info = config['version_info']
bundle_info = config['bundle']

def build_csv():
    csv = load_yaml_file(csv_base)
    role = load_yaml_file(role_file)
    operator = load_yaml_file(operator_dep_file)
    service_account = load_yaml_file(service_account_file)

    install_dep_item = {
        'name': get_name(operator),
        'spec': operator['spec']
    }
    # update operator image version tag
    install_dep_item['spec']['template']['spec']['containers'][0]['image'] = 'nvmesh-operator:{version}-{release}'.format(**version_info)

    cluster_permissions = {
        'serviceAccountName': get_name(service_account),
        'rules': role['rules']
    }

    version = version_info['version']


    csv['metadata']['name'] = 'nvmesh-operator.%s' % (version)
    csv['spec']['version'] = "{version}-{release}".format(**version_info)
    csv['spec']['install']['spec']['deployments'] = [install_dep_item]
    csv['spec']['install']['spec']['clusterPermissions'] = [cluster_permissions]


    write_yaml_file(csv, output_file)

    print("ClusterServiceVersion file generated at %s" % output_file)

def copy_and_format_crd():
    crd = load_yaml_file(crd_base)
    del crd['metadata']['creationTimestamp']
    write_yaml_file(crd, crd_base)

def build_deploy_dir():
    copyfile(crd_base, path.join(deploy, "010_nvmesh_crd.yaml"))
    copyfile(path.join(bases, "extra/service_account.yaml"), path.join(deploy, "020_service_account.yaml"))
    copyfile(path.join(bases, "rbac/role.yaml"), path.join(deploy, "030_role.yaml"))
    copyfile(path.join(bases, "rbac/role_binding.yaml"), path.join(deploy, "040_role_binding.yaml"))
    copyfile(path.join(bases, "operator/deployment.yaml"), path.join(deploy, "050_operator-deployment.yaml"))

    rmtree(path.join(deploy, "samples"))
    copytree(path.join(bases, "samples"), path.join(deploy, "samples"))

def build_bundle_dir():
    build_csv()
    copyfile(path.join(bases, "crd/nvmesh.crd.yaml"), path.join(bundle,"manifests", "nvmesh_crd.yaml"))

def update_catalog_source():
    catalog_source_file = '../operator-hub/dev/catalog_source.yaml'
    cat_source = load_yaml_file(catalog_source_file)
    image = '{image}:{version}-{rel}-{bundle_build}'.format(
        image=bundle_info['dev']['index_image_name'],
        version=version_info['version'],
        rel=version_info['release'],
        bundle_build=bundle_info['dev']['bundle_build']
    )
    cat_source['spec']['image'] = image

    write_yaml_file(cat_source, catalog_source_file)

copy_and_format_crd()
build_deploy_dir()
build_bundle_dir()
update_catalog_source()