# Start infinite loop process with KubeCluster in it

import asyncio
import click
import os
from dask_kubernetes import KubeCluster, KubeConfig, InCluster, make_pod_from_dict
import yaml
import logging

logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)

"""
This function authorizes with kubernetes and creates KubeCluster object.
The worker pod yaml file must be ready at this point.
"""


async def start_dask_kube_cluster(**params):

    name = params.get("name", os.environ.get("DASK_NAME", None))  # unique name of the cluster
    namespace = params.get(
        "namespace", os.environ.get("DASK_NAMESPACE", "community-namespace")
    )  # dask namespace
    worker_config = params.get("worker_config", "/etc/config/worker_spec.yaml")
    starting_workers = int(
        params.get("starting_workers", os.environ.get("DASK_STARTING_WORKERS", 2))
    )  # starting number of worker pods
    max_workers = int(
        params.get("max_workers", os.environ.get("DASK_MAX_WORKERS", 10))
    )  # max number of worker pods for this cluster
    scheduler_timeout_min = int(
        params.get("scheduler_timeout_min", os.environ.get("DASK_WORKER_TOUT", 9999))
    )  # time (min) the cluster stays active

    kube_config_file = params.get("kube_config_file")
    if name is None or len(name) == 0:
        log.exception("Name of the cluster must be set and not empty")
        raise Exception("Critical error")

    log.info(f"Starting KubeCluster for {name} in namespace {namespace}, params:\n {params} ")
    max_workers = int(max_workers)
    starting_workers = max(2, int(starting_workers))

    if kube_config_file is not None:
        kube_config = KubeConfig(os.path.expanduser(kube_config_file))
        await kube_config.load()  # kubernetes authentication
    else:
        kube_auth = InCluster()  # with ServiceAccount
        await kube_auth.load()  # kubernetes authentication

    pod_template = None
    if worker_config is None or not os.path.exists(worker_config):
        log.exception(f"Could not create dask worker config, filename {worker_config}")
        raise Exception("Critical error")

    with open(worker_config) as f:
        d = yaml.safe_load(f)
        pod_template = make_pod_from_dict(d)

    log.debug(f"\nPod template:\n{pod_template}\n")

    cluster = KubeCluster(
        pod_template=pod_template,
        deploy_mode="remote",
        name=name,
        namespace=namespace,
        scheduler_timeout=f"{scheduler_timeout_min} m",
    )

    if starting_workers > 2:
        log.info(f"Starting scaled up to {starting_workers} workers, max is {max_workers}")
        cluster.scale_up(starting_workers)  # specify number of nodes explicitly

    cluster.adapt(
        minimum=starting_workers, maximum=max_workers
    )  # dynamically scale based on current workload
    return cluster


# command line interface
@click.group()
def cli():
    pass


"""  Start infinite async loop for KubeCluster. It will start scheduler pod and worker pods."""


@cli.command()
@click.argument("name")  # cluster name
@click.option("--namespace")  # dask cluster namespace
@click.option("--worker-config")  # worker config filename
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
        log.info(f"Using kube config file: {kube_config_file}")

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
    log.info(f"\nDask cluster {name} in namespace {namespace}")
    log.info(f"\tworker config: {worker_config}")
    log.info(f"\tworkers: start: {starting_workers}, max: {max_workers}")
    log.info(f"\tcpus per pod {cpus_per_pod}")
    log.info(f"\tscheduler timeout: {scheduler_timeout_min} min")
    log.info(f"\tscheduler address: {result.scheduler_address}")

    loop.run_forever()
    loop.close()


if __name__ == "__main__":  # main stub
    cli()
