
__filename__=$0

pod_name=""
namespace="default"
since=""
pod_logs="false"
trace="false"

show_help() {
    echo "Usage: $__filename__ [--trace] [--logs] --pod <pod-name> --namespace <namespace>"
    echo ""
    echo "--logs                collect pod logs and output to <pod_name>.log"
    echo "--trace               collect trace logs from within the pod and output to stdout"
    echo "--pod                 the name of pod containing toma's container"
    echo "-n|--namespace        the namespace where the pod is deployed. deafult value is \"default\""
    echo "--since               set value for pager.py --since flag i.e: \"--since now-120s\""
}

parse_args() {
    while [[ $# -gt 0 ]]
    do
    key="$1"

    case $key in
        --trace)
            trace="true"
            shift
        ;;
        --pod-logs)
            pod_logs="true"
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
    echo "Error: Missing pod name"
    show_help
    exit 1
fi

since_with_flag=""
if [ ! -z "$since" ]; then
    since_with_flag="--since $since"
fi

if [ "$pod_logs" == "true" ]; then
    echo "running: kubectl exec -n $namespace $pod_name -c toma > $pod_name.log"
    kubectl logs -n $namespace $pod_name -c toma > $pod_name.log
fi

if [ "$trace" == "true" ]; then
    echo "running: kubectl exec -n $namespace $pod_name -c toma -- /bin/bash -c \"cd /var/log/NVMesh/trace_daemon/ ; ./pager.py --toma $since_with_flag\""
    kubectl exec -n $namespace $pod_name -c toma -- /bin/bash -c "cd /var/log/NVMesh/trace_daemon/ ; ./pager.py --toma $since_with_flag"
fi