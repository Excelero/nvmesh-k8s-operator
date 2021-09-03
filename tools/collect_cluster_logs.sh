
__filename__=$0

pod_name=""
namespace="default"
log_type="pod"
hostname=""
all_from_host=""
debug_enabled=false

COMPONENTS=(
    toma
    tracer
    client
    target
    mcs
    agent
    csi-node-driver
    csi-controller
    management
)

log() {
    level=$1
    shift

    if [ "$level" == "error" ]; then
        echo "$level - $@" >&2
    elif [ "$level" == "debug" ]; then
        if [ "$debug_enabled" == "true" ]; then
            echo "$level - $@"
        fi
    elif [ "$level" == "info" ]; then
        echo "$level - $@"
    else
        echo "Unknown log level $level"
        exit 3
    fi
}

print_code() {
    echo -e "\033[0;35m$@\033[0m"
}

print_comment() {
    echo -e "\033[1;37m$@\033[0m"
}



show_help() {
    echo "Usage: $__filename__ [--log-type <trace|pod>] [--host <hostname>] [--pod <pod-name>] [--namespace <namespace>]"
    echo ""
    echo "-t|--trace        collect trace_daemon logs from within the pod and output to stdout."
    echo "--pod             collect logs from a specific pod. otherwise the pod name will be determined using the --host"
    echo "-c|--component    when --log-type=pod this will determine which container logs to collect"
    echo "-n|--namespace    the namespace where the pod is deployed. deafult value is \"default\""
    echo "--host            the kubernetes name of the host on which the toma is running (try running kubectl get node -o name)"
    echo "--debug           enable debug logs for this script"



    echo ""
    echo ""
    echo ""

    # Exmaples
    print_comment "Examples:"
    script_name=$0
    print_comment "Collect all logs from the cluster (including management, trace_daemon logs and ConfigMaps)"
    print_code "${script_name} --all"
    echo ""

    print_comment "Collect all client and target logs from node 'worker1.excelero.com'"
    print_code "${script_name} --all-from-host worker1.excelero.com"
    echo ""


    print_comment "Collect pod logs for csi-node-driver on 'worker1.excelero.com'"
    print_code "${script_name} --host master1.excelero.com -c csi-node-driver"
    print_comment "Available components are: ${COMPONENTS[@]}"
    echo ""

    print_comment "Collect trace_daemon logs for toma from master1.excelero.com for the last 2 minutes"
    print_code "${script_name} --trace --host master1.excelero.com -- --toma --since now-120"
    print_comment "for more pager.py options run:"
    print_code "${script_name} --trace --host master1.excelero.com -- --help"

    echo ""

}

parse_args() {
    while [[ $# -gt 0 ]]
    do
    key="$1"
    case $key in
        -t|--trace)
            log_type="trace"
            component="trace"
            shift
        ;;
        --debug)
            debug_enabled="true"
            log "debug" "Debug enabled"
            shift
        ;;
        --all-from-host)
            host="$2"
            all_from_host="true"
            shift
            shift
        ;;
        --all)
            all_logs="true"
            shift
        ;;
        --host)
            host="$2"
            shift
            shift
        ;;
        --pod)
            pod_name="$2"
            shift
            shift
        ;;
        -c|--component)
            component="$2"
            shift
            shift
        ;;
        -n|--namespace)
            namespace="$2"
            shift
            shift
        ;;
        --)
            shift
            pager_args="$@"
            break
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

get_pod_and_container_names() {
    if [ -z "$pod_name" ]; then
        if [ -z "$component" ]; then
            log "error" "missing --component flag"
            show_help
            exit 1
        elif [ -z "$host" ]; then
            log "error" "missing --host flag"
        else
            # get pod name from hostname provided
            if [ "$log_type" == "trace" ]; then
                label_selector="nvmesh.excelero.com/component=client"
            elif [ "$log_type" == "pod" ]; then
                if [ "$component" == "tracer" ]; then
                    label_selector="nvmesh.excelero.com/component=client"
                    container="tracer"
                elif [ "$component" == "client" ]; then
                    label_selector="nvmesh.excelero.com/component=client"
                    container="driver-container"
                elif [ "$component" == "target" ]; then
                    label_selector="nvmesh.excelero.com/component=target"
                    container="driver-container"
                elif [ "$component" == "toma" ]; then
                    label_selector="nvmesh.excelero.com/component=target"
                    container="toma"
                elif [ "$component" == "mcs" ] || [ "$component" == "agent" ]; then
                    label_selector="nvmesh.excelero.com/component=mcs-agent"
                    container="$component"
                elif [ "$component" == "csi-node-driver" ]; then
                    label_selector="nvmesh.excelero.com/component=csi-node-driver"
                    container="nvmesh-csi-driver"
                elif [ "$component" == "csi-controller" ]; then
                    label_selector="nvmesh.excelero.com/component=csi-controller"
                    container="nvmesh-csi-controller"
                elif [ "$component" == "management" ]; then
                    label_selector="nvmesh.excelero.com/component=nvmesh-management"
                    pod_name="nvmesh-management-0"
                    container="nvmesh-management"
                fi
            fi

            pod_name=$(kubectl get pods -o name --field-selector spec.nodeName=$host --selector=${label_selector} | head -1)

            if [ -z "$pod_name" ]; then
                if [ -z "$all_logs" ] && [ -z "$all_from_host" ]; then
                    # if this is a multi logs command ignore, otherwise log as error and exit with error
                    log "error" "Error: could not find the pod for ${label_selector} that runs on host ${host}"
                    exit 2
                fi

            else
                log "debug" "Found pod name: $pod_name" >&2
            fi
        fi
    fi
}

collect_logs() {
    log "debug" "collect_logs with pod_name=$pod_name component=$component"
    # populates `pod_name` and `container`
    get_pod_and_container_names

    if [ -z "$pod_name" ]; then
        return
    fi

    log "info" "Collecting logs from $component on $host"

    # convert pod/container_name to pod-container_name
    safe_pod_name=$(echo $pod_name | sed -e 's/\//-/g')

    if [ "$log_type" == "pod" ]; then
        output_file="${file_prefix}${host}_${safe_pod_name}_${container}.log"
        log "debug" "running: kubectl logs -n $namespace $pod_name -c $container > ${pod_name}_${container}.log"
        kubectl logs -n $namespace $pod_name -c $container > $output_file
    elif [ "$log_type" == "trace" ]; then
        log "debug" "pager.py args=$pager_args"
        output_file="${file_prefix}${host}_${safe_pod_name}_trace_daemon.log"
        redirect=
        log "debug" "running: kubectl exec -n $namespace $pod_name -c tracer -- /bin/bash -c \"cd /var/log/NVMesh/trace_daemon/ ; ./pager.py $pager_args\""
        kubectl exec -n $namespace $pod_name -c tracer -- /bin/bash -c "cd /var/log/NVMesh/trace_daemon/ ; ./pager.py $pager_args" > $output_file
    else
        log "error" "Unknown log type $log_type"
        show_help
        exit 1
    fi
}

collect_all_from_host() {
    log "info" "Collecting logs from Node $host"

    file_prefix="${log_dir_name}/${host}/"
    mkdir -p "$file_prefix"

    log_type="pod"
    components=(
        toma
        tracer
        client
        target
        mcs
        agent
        csi-node-driver
        csi-controller
    )

    for c in ${components[@]}; do
        pod_name=""
        component=$c
        collect_logs
    done

    log "info" "Collecting trace_daemon logs from $host"
    pod_name=""
    log_type="trace"
    collect_logs
}

collect_config_maps() {
    log "info" "Collecting ConfigMaps"

    configmaps=(
        mongo-conf
        nvmesh-core-config
        nvmesh-csi-config
        nvmesh-csi-topology
        nvmesh-mgmt-config
    )

    cm_dir="${log_dir_name}/config_maps"
    mkdir -p $cm_dir

    for cm_name in ${configmaps[@]}; do
        log "info" "Collecting ConfigMap $cm_name"
        kubectl get configmap $cm_name -o yaml > "${cm_dir}/${cm_name}"
    done
}

collect_management_logs() {
    log "info" "Collecting Management logs"
    mgmt_nodes=$(kubectl get pod -l app=nvmesh-management -o=jsonpath='{.items[*].spec.nodeName}')
    log "debug" "Found management nodes: ${mgmt_nodes}"

    file_prefix="${log_dir_name}/management/"
    mkdir -p $file_prefix

    log_type="pod"
    for node in $mgmt_nodes; do
        pod_name=""
        host="$node"
        component=management
        collect_logs
    done
}

collect_csi_controller_logs() {
    log "info" "Collecting CSI Controller logs"
    csi_ctrl_nodes=$(kubectl get pod -l nvmesh.excelero.com/component=csi-controller -o=jsonpath='{.items[*].spec.nodeName}')
    log "debug" "Found CSI Controller nodes: ${csi_ctrl_nodes}"

    file_prefix="${log_dir_name}/csi-controller/"
    mkdir -p $file_prefix

    log_type="pod"
    for node in $csi_ctrl_nodes; do
        pod_name=""
        host="$node"
        component=csi-controller
        collect_logs
    done
}


get_core_nodes() {
    client_nodes=$(kubectl get pod -l "nvmesh.excelero.com/component=client" -o=jsonpath='{.items[*].spec.nodeName}')
    target_nodes=$(kubectl get pod -l "nvmesh.excelero.com/component=target" -o=jsonpath='{.items[*].spec.nodeName}')

    declare -A node_type
    core_nodes=()
    for n in $client_nodes; do
        core_nodes+=( $n )
        node_type[$n]="client-only"
    done

    for n in $target_nodes; do
        if [ -z "${node_type[$n]}" ]; then
            core_nodes+=( $n )
        fi

        node_type[$n]="client-target"
    done

    if [ "$debug_enabled" == "true" ]; then
        for node in "${!node_type[@]}"; do
            log "debug" "node $node is ${node_type[$node]}"
        done
    fi

    log "debug" "core_nodes=${core_nodes[@]}"
}

collect_all_logs() {
    log "info" "Collecting all logs"

    get_core_nodes
    log "debug" "Found client and target nodes: ${core_nodes[@]}"

    for node in "${core_nodes[@]}"; do
        host=$node
        collect_all_from_host
    done

    collect_management_logs

    collect_csi_controller_logs

    collect_config_maps

    log "info" "Finished collecting all logs"
}

get_dir_name_with_date() {
    echo "log_from_$(date '+%F-%T')"
}

################## Main ##################

parse_args $@

if [ "$all_from_host" == "true" ]; then
    log_dir_name=$(get_dir_name_with_date)
    collect_all_from_host
    log "info" "Logs are available at $log_dir_name"
elif [ "$all_logs" == "true" ]; then
    log_dir_name=$(get_dir_name_with_date)
    collect_all_logs
    log "info" "Logs are available at $log_dir_name"
else
    collect_logs
fi
