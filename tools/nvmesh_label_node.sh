#!/bin/bash

__filename__=$0

type=""
all_nodes="false"
nodes=()

print_err() {
    echo $1 >&2
}

show_help() {
    echo "Usage: $__filename__ [--node <node name>] [--all] [--type client|target|both|mgmt]"
    echo ""
    echo "-a|--all     label all worker nodes"
    echo "-n|--node    the node to set labels on"
    echo "-t|--type    type of nvmesh node - on of the following: client|target|both|mgmt"
    echo "-r|--remove  if set the selected nvmesh labels will be removed from this node"
}

parse_args() {
    while [[ $# -gt 0 ]]
    do
    key="$1"

    case $key in
        -a|--all)
            all_nodes="true"
            shift
        ;;
        -n|--node)
            nodes+=( $2 )
            shift
            shift
        ;;
        -t|--type)
            type="$2"
            shift
            shift
        ;;
        -r|--remove)
            remove="true"
            shift
        ;;
        -h|--help)
            show_help
            exit 0
        ;;
        *)  # unknown option
            echo "Unknown option $key"
            show_help
            exit 1
        ;;
    esac
    done

}

validate_args() {
    num_of_nodes=${#nodes[@]}
    if [ $num_of_nodes -eq 0 ] && [ $all_nodes != "true" ]; then
        print_err "Error: Either -n <node> or --all must be specified"
        show_help
        exit 1
    fi

    if [ "$type" == "" ]; then
        print_err "Error: Missing --type argument"
        show_help
        exit 1
    fi

    if [ "$type" != "client" ] && [ "$type" != "target" ] && [ "$type" != "both" ] && [ "$type" != "mgmt" ]; then
        print_err "Error: Unknown type $type. supported values are: client, target or both."
        show_help
        exit 1
    fi
}

check_err() {
    if [ $1 -ne 0 ]; then
        print_err "Error: $2"
        exit 2
    fi
}

get_nvmesh_label() {
    if [ "$remove" == "true" ]; then
        label="nvmesh.excelero.com/$1-"
        echo "removing nvmesh labels from node $n"
    else
        label=$(printf %q nvmesh.excelero.com/$1="")
        echo "labeling node $n"
    fi
}

################## Main ##################

parse_args $@

validate_args

if [ "$all_nodes" == "true" ]; then
    openshift_workers_labels="-l node-role.kubernetes.io/worker"
    nodes=$(kubectl get nodes $openshift_workers_labels -o=custom-columns=NAME:.metadata.name --no-headers)

    if [ "$nodes" == "" ]; then
        echo "Failed to find nodes on the cluster"
    fi
fi

for n in $nodes; do
    if [ "$type" == "client" ] || [ "$type" == "both" ]; then
        get_nvmesh_label nvmesh-client
        kubectl label node $n "$label"
        check_err $? "failed to label node $n with $label."
    fi

    if [ "$type" == "target" ] || [ "$type" == "both" ]; then
        get_nvmesh_label nvmesh-target
        kubectl label node $n "$label"
        check_err $? "failed to label node $n with $label"
    fi

    if [ "$type" == "mgmt" ]; then
        get_nvmesh_label nvmesh-management
        kubectl label node $n "$label"
        check_err $? "failed to label node $n with $label"
    fi
done

echo "Done."