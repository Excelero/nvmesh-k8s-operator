#!/usr/bin/env python2

import json
import yaml
import sys
from os import path
import pathlib
from shutil import copyfile, copytree, rmtree

output_file = "../operator-hub/catalog_bundle/manifests/cluster_service_version.yaml"
config_file = "./config.yaml"

bases = "./bases"
deploy = "../deploy"
operator_hub_dir = "../operator-hub"
bundle_dir = path.join(operator_hub_dir,"catalog_bundle")

crd_base = path.join(bases, "crd/nvmesh.crd.yaml")
csv_base = path.join(bases, "csv/csv.yaml")
role_file = path.join(bases, "rbac/role.yaml")
operator_dep_file = path.join(bases, "operator/deployment.yaml")
service_account_file = path.join(bases, "extra/service_account.yaml")
cr_sample_source_file = path.join(bases, "samples/nvmesh/nvmesh_v1.yaml")
cr_sample_for_testing_yaml = path.join(operator_hub_dir, "dev/cr_sample.yaml")
generated_yaml_file_comment = [
    "# DO NOT EDIT THIS FILE\n",
    "# This file is auto-generated by manifests/build_manifests.py\n"
]

def load_yaml_file(filename):
    with open(filename, 'r') as f:
        return yaml.safe_load(f)

def write_yaml_file(obj, output_file):
    with open(output_file, 'w') as f:
        f.writelines(generated_yaml_file_comment)
        yaml.dump(obj, f, sort_keys=True)

def get_name(obj):
    return obj['metadata']['name']

config = load_yaml_file(config_file)
version_info = config['operator']
bundle_info = config['bundle']

def get_alm_examples():
    alm_example = bundle_info["alm-example"]

    # add object status for scorecard validation
    alm_example['status'] = {
        'reconcileStatus': {},
        'actionsStatus': {}
    }

    # add skipUninstall to alm_example
    #cr_sample['spec']['operator'] = { 'skipUninstall': True }

    write_yaml_file(alm_example, cr_sample_for_testing_yaml)
    alm_example_as_json_string = json.dumps(alm_example, separators=(',', ':'))
    alm_examples = '[{}]'.format(alm_example_as_json_string)
    return alm_examples

def get_bundle_name():
	bundle_version = bundle_info['version']
	return 'nvmesh-operator.%s' % (bundle_version)

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
    operator_image = get_operator_image()
    operatorPodSpec = install_dep_item['spec']['template']['spec']
    operatorContainer = operatorPodSpec['containers'][0]
    operatorContainer['image'] = operator_image
    operatorContainer['args'].append("--openshift")
    operatorContainer['args'].append("--core-image-tag")
    operatorContainer['args'].append(version_info["core_image_tag"])

    cluster_permissions = {
        'serviceAccountName': get_name(service_account),
        'rules': role['rules']
    }

    csv['metadata']['name'] = get_bundle_name()
    csv['metadata']['annotations']['alm-examples'] = get_alm_examples()
    csv['metadata']['annotations']['containerImage'] = operator_image

    csv['spec']['version'] = version_info['version']
    csv['spec']['install']['spec']['deployments'] = [install_dep_item]
    csv['spec']['install']['spec']['clusterPermissions'] = [cluster_permissions]

    write_yaml_file(csv, output_file)

    print("ClusterServiceVersion file generated at %s" % output_file)

def copy_and_format_crd():
    crd = load_yaml_file(crd_base)
    del crd['metadata']['creationTimestamp']
    write_yaml_file(crd, crd_base)

def get_operator_image(repo=None):
    ver_info_copy = version_info.copy()
    if repo:
        ver_info_copy['repo'] = repo

    return '{repo}/{image_name}:{version}-{release}'.format(**ver_info_copy)

def get_deployment_for_kubernetes():
    deployment = load_yaml_file(operator_dep_file)

    operatorPodSpec = deployment['spec']['template']['spec']
    operatorContainer = operatorPodSpec['containers'][0]

    # For the kubectl deploy yamls we will use the image from docker hub
    operatorContainer['image'] = get_operator_image('excelero')
    return deployment

def build_deploy_dir():
    copyfile(crd_base, path.join(deploy, "010_nvmesh_crd.yaml"))
    copyfile(path.join(bases, "extra/service_account.yaml"), path.join(deploy, "020_service_account.yaml"))
    copyfile(path.join(bases, "rbac/role.yaml"), path.join(deploy, "030_role.yaml"))
    copyfile(path.join(bases, "rbac/role_binding.yaml"), path.join(deploy, "040_role_binding.yaml"))

    deployment = get_deployment_for_kubernetes()
    write_yaml_file(deployment, path.join(deploy, "050_operator-deployment.yaml"))

    rmtree(path.join(deploy, "samples"))
    copytree(path.join(bases, "samples"), path.join(deploy, "samples"))

def build_bundle_dir():
    build_csv()
    copyfile(path.join(bases, "crd/nvmesh.crd.yaml"), path.join(bundle_dir,"manifests", "nvmesh_crd.yaml"))

def update_catalog_source():
    catalog_source_file = path.join(operator_hub_dir, "dev/catalog_source.yaml")
    cat_source = load_yaml_file(catalog_source_file)
    image = '{image}:{version}-{rel}'.format(
        image=bundle_info['dev']['index_image_name'],
        version=bundle_info['version'],
        rel=bundle_info['release']
    )
    cat_source['spec']['image'] = image
    cat_source['metadata']['name'] = 'nvmesh-catalog-{}'.format(bundle_info['version'])
    write_yaml_file(cat_source, catalog_source_file)

def update_subscription():
    subscription_file = path.join(operator_hub_dir, "dev/subscription.yaml")
    subscription = load_yaml_file(subscription_file)
    subscription['spec']['startingCSV'] = get_bundle_name()
    write_yaml_file(subscription, subscription_file)

copy_and_format_crd()
build_deploy_dir()
build_bundle_dir()
update_catalog_source()
update_subscription()