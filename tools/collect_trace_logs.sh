
__filename__=$0

pod_name=""
namespace="default"
since=""
log_type="trace"
hostname=""
print_to_stdout="false"
redirect=""

print_err() {
    echo $1 >&2
}

show_help() {
    echo "Usage: $__filename__ [--log-type <trace|pod>] [--host <hostname>] [--pod <pod-name>] [--namespace <namespace>]"
    echo ""
    echo "-t|--log-type     ( pod | trace ) \"pod\" for the pod logs, \"trace\" will collect trace logs from within the pod and output to stdout. default is \"trace\""
    echo "--pod             the name of pod containing toma's container"
    echo "-n|--namespace    the namespace where the pod is deployed. deafult value is \"default\""
    echo "--since           set value for pager.py --since flag i.e: \"--since now-120s\""
    echo "--host            the kubernetes name of the host on which the toma is running (try running kubectl get node -o name)"
    echo "-o|--stdout          stream the output to stdouyt instead of writing to a file"
}

parse_args() {
    while [[ $# -gt 0 ]]
    do
    key="$1"

    case $key in
        -t|--log-type)
            log_type="$2"
            shift
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
        -n|--namespace)
            namespace="$2"
            shift
            shift
        ;;
        --since)
            since="$2"
            shift
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

################## Main ##################

parse_args $@

if [ -z "$pod_name" ]; then
    if [ -z "$host" ]; then
        print_err "Error: Missing pod name"
        show_help
        exit 1
    else
        # get pod anme from hostname provided
        pod_name=$(kubectl get pods -o name --field-selector spec.nodeName=$host --selector=name=nvmesh-target-driver-container | head -1)
        if [ -z "$pod_name" ]; then
            print_err "Error: could not find the pod nvmesh-target-driver-container that runs on host $host"
            exit 2
        fi

        echo "Found pod name: $pod_name" >&2
    fi
fi

since_with_flag=""
if [ ! -z "$since" ]; then
    since_with_flag="--since $since"
fi

if [ "$log_type" == "pod" ]; then
    echo "running: kubectl exec -n $namespace $pod_name -c toma > $pod_name.log"
    kubectl logs -n $namespace $pod_name -c toma
elif [ "$log_type" == "trace" ]; then
    echo "running: kubectl exec -n $namespace $pod_name -c toma -- /bin/bash -c \"cd /var/log/NVMesh/trace_daemon/ ; ./pager.py --toma $since_with_flag\""
    kubectl exec -n $namespace $pod_name -c toma -- /bin/bash -c "cd /var/log/NVMesh/trace_daemon/ ; ./pager.py --toma $since_with_flag"
else
    print_err "Unknown log type $log_type"
    show_help
    exit 1
fi
