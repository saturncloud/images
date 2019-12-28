# starts infinite loop process with KubeCluster in it

import asyncio
import click
import os
from dask_kubernetes import KubeCluster, KubeConfig, InCluster, make_pod_from_dict
import yaml


"""
This function authorizes with kubernetes and creates KubeCluster object.
The worker pod yaml file must be ready at this point.
"""


async def start_dask_kube_cluster(**params):

    name = params.get("name", os.environ.get("DASK_NAME", None))  ## unique name of the cluster
    namespace = params.get(
        "namespace", os.environ.get("DASK_NAMESPACE", "community-namespace")
    )  ## dask namespace
    worker_config = params.get("worker_config", "/etc/config/worker_spec.yaml")
    starting_workers = int(
        params.get("starting_workers", os.environ.get("DASK_STARTING_WORKERS", 2))
    )  ## starting number of worker pods
    max_workers = int(
        params.get("max_workers", os.environ.get("DASK_MAX_WORKERS", 10))
    )  ## max number of worker pods for this cluster
    cpus_per_pod = float(
        params.get("cpus_per_pod", os.environ.get("DASK_CPU_PER_POD", 1))
    )  ## threads on a single pod
    scheduler_timeout_min = int(
        params.get("scheduler_timeout_min", os.environ.get("DASK_WORKER_TOUT", 9999))
    )  ## time (min) the cluster stays active

    kube_config_file = params.get("kube_config_file")  # TODO: remove when local testing is done
    assert name is not None and len(name) > 0, "Name of the cluster must be set and not empty!"

    print(f"Starting KubeCluster for {name} in namespace {namespace}, params: ", params)
    max_workers = int(max_workers)
    starting_workers = max(2, int(starting_workers))

    if kube_config_file is not None:
        kube_config = KubeConfig(os.path.expanduser(kube_config_file))
        await kube_config.load()  # kubernetes authentication
    else:
        kube_auth = InCluster()  # with ServiceAccount
        await kube_auth.load()  # kubernetes authentication

    pod_template = None
    assert (
        worker_config is not None
    ), "[Dask Cluster] Critical error: could not create worker config"
    with open(worker_config) as f:
        d = yaml.safe_load(f)
        pod_template = make_pod_from_dict(d)

    print(pod_template)  # a lot of interesting undocumented stuff there

    cluster = KubeCluster(
        pod_template=pod_template,
        deploy_mode="remote",
        name=name,
        namespace=namespace,
        scheduler_timeout="%d m" % scheduler_timeout_min,
    )

    if starting_workers > 2:
        print("Starting scaled up to %s workers" % starting_workers)
        cluster.scale_up(starting_workers)  # specify number of nodes explicitly

    cluster.adapt(
        minimum=starting_workers, maximum=max_workers
    )  # dynamically scale based on current workload
    return cluster


## command line interface
@click.group()
@click.pass_context
def cli(ctx):
    if ctx.invoked_subcommand is None:
        print("the command you need is:  'kube-cluster'")


"""  Starts infinite async loop for KubeCluster. 
     It will start scheduler pod and worker pods.     
"""


@cli.command()
@click.argument("name")  ## cluster name
@click.option("--namespace")  ## dask cluster namespace
@click.option("--worker-config")  ## worker config filename
@click.option("--starting-workers")
@click.option("--max-workers")
@click.option("--cpus-per-pod")
@click.option("--scheduler-timeout-min")
@click.option("--use-kube-config", default=False)
def kube_cluster(
    name,
    namespace,
    worker_config,
    starting_workers,
    max_workers,
    cpus_per_pod,
    scheduler_timeout_min,
    use_kube_config,
):
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)

    kube_config_file = "~/.kube/config" if use_kube_config else None
    if kube_config_file is not None:
        print("Using kube config file:", kube_config_file)

    params = {"name": name}

    if namespace is not None:
        params["namespace"] = namespace
    if worker_config is not None:
        params["worker_config"] = worker_config
    if starting_workers is not None:
        params["starting_workers"] = starting_workers
    if max_workers is not None:
        params["max_workers"] = max_workers
    if cpus_per_pod is not None:
        params["cpus_per_pod"] = cpus_per_pod
    if kube_config_file is not None:
        params["kube_config_file"] = kube_config_file
    if scheduler_timeout_min is not None:
        params["scheduler_timeout_min"] = scheduler_timeout_min

    result = loop.run_until_complete(start_dask_kube_cluster(**params))
    print(f"\nDask cluster {name} started in namespace {namespace}")
    print(f"\tworker config: {worker_config}")
    print(f"\tworkers: start: {starting_workers}, max: {max_workers}")
    print(f"\tcpus per pod {cpus_per_pod}")
    print(f"\tscheduler timeout: {scheduler_timeout_min} min")
    print(f"\tscheduler address: {result.scheduler_address}")
    loop.run_forever()
    loop.close()


if __name__ == "__main__":  # main stub
    cli()
