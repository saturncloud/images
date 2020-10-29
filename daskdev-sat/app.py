import json
import logging
import os
import yaml
import asyncio
import kubernetes

import tornado.ioloop
from tornado.web import RequestHandler, Application

from distributed.core import rpc as dask_rpc
from distributed.comm import resolve_address
from distributed.worker import get_client
from dask_kubernetes import KubeCluster, make_pod_from_dict
from dask_kubernetes.core import Scheduler, SCHEDULER_PORT


logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)


NAME = os.environ.get("DASK_NAME")
NAMESPACE = os.environ.get("DASK_NAMESPACE", "main-namespace")
DASHBOARD_LINK = os.environ.get("DASK_DASHBOARD_LINK", None)
N_WORKERS = int(os.environ.get("DASK_N_WORKERS", 0))
WORKER_CONFIG = "/etc/config/worker_spec.yaml"
SCHEDULER_CONFIG = "/etc/config/scheduler_spec.yaml"


def make_cluster(n_workers):
    with open(WORKER_CONFIG) as f:
        d = yaml.safe_load(f)
        pod_template = make_pod_from_dict(d)
    with open(SCHEDULER_CONFIG) as f:
        d = yaml.safe_load(f)
        scheduler_pod_template = make_pod_from_dict(d)

    log.info(f"Starting dask cluster {NAME} in namespace {NAMESPACE}")
    log.info(f"n_workers: {N_WORKERS}")
    log.info(f"worker config: {WORKER_CONFIG}")
    log.info(f"scheduler config: {SCHEDULER_CONFIG}")
    log.info(f"dashboard address: {DASHBOARD_LINK}")
    log.info(f"distributed version: {distributed.__version__}")

    return SaturnKubeCluster(
        n_workers=n_workers,
        pod_template=pod_template,
        scheduler_pod_template=scheduler_pod_template,
        deploy_mode="remote",
        name=NAME,
        namespace=NAMESPACE,
        dashboard_link=DASHBOARD_LINK,
    )


class SaturnSetup:
    name = "saturn_setup"

    def __init__(self, scheduler_address=None):
        self.scheduler_address = scheduler_address

    def setup(self, worker=None):
        worker.scheduler.addr = resolve_address(self.scheduler_address)


class SaturnKubeCluster(KubeCluster):
    """Class that inherits from dask-kubernetes cluster"""

    def __init__(self, *args, dashboard_link=None, **kwargs):
        """Init as usual, but add dashboard_link to object and start workers"""
        super().__init__(*args, **kwargs)

        self._dashboard_link = dashboard_link

    async def _start(self):
        log.info("Starting scheduler")
        if self._n_workers > 0:
            name = self._generate_name
            namespace = self._namespace
            scheduler_address = f"tcp://{name}.{namespace}:{SCHEDULER_PORT}"
            self.scheduler = Scheduler(
                cluster=self,
                idle_timeout=self._idle_timeout,
                service_wait_timeout_s=self._scheduler_service_wait_timeout,
                core_api=kubernetes.client.CoreV1Api(),
                pod_template=self.scheduler_pod_template,
                namespace=namespace,
            )
            self.scheduler.address = scheduler_address
            self.scheduler_comm = dask_rpc(
                scheduler_address,
                connection_args=self.security.get_connection_args("client"),
            )
            self.pod_template.spec.containers[0].env.append(
                kubernetes.client.V1EnvVar(
                    name="SATURN_DASK_SCHEDULER_ADDRESS", value=scheduler_address
                )
            )
            self._lock = asyncio.Lock()
            log.info("Starting workers")
            asyncio.gather(self._correct_state(), super()._start())
        else:
            await super()._start()

    @property
    def dashboard_link(self):
        """Overwrite base class and just return attr"""
        return self._dashboard_link


class SaturnClusterApplication(Application):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.cluster = make_cluster(n_workers=N_WORKERS)
        log.info("Cluster successfully created in tornado!")
        log.info(f"Scheduler: {self.cluster.scheduler_address}")
        log.info(f"Dashboard: {self.cluster.dashboard_link}")


class MainHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(str(cluster))


class InfoHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        info = {
            "scheduler_address": cluster.scheduler_address,
            "dashboard_link": cluster.dashboard_link,
        }
        self.write(json.dumps(info))


class RegisterPluginHandler(RequestHandler):
    def post(self):
        log.info("Registering default plugins")
        cluster = self.application.cluster
        scheduler_address = cluster.scheduler_address
        with get_client(scheduler_address) as client:
            output = client.register_worker_plugin(SaturnSetup(scheduler_address=scheduler_address))
            log.info(output)


class SchedulerInfoHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(json.dumps(cluster.scheduler_info))


class StatusHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(json.dumps({"status": cluster.status.value}))


class ScaleHandler(RequestHandler):
    def post(self):
        cluster = self.application.cluster
        if hasattr(cluster, "_adaptive"):
            cluster._adaptive.stop()
        body = json.loads(self.request.body)
        log.info(f"Scaling cluster: {body}")
        cluster.scale(**body)


class AdaptHandler(RequestHandler):
    def post(self):
        cluster = self.application.cluster
        body = json.loads(self.request.body)
        log.info(f"Adapting cluster: {body}")
        cluster.adapt(**body)


def make_app():
    return SaturnClusterApplication(
        [
            (r"/", MainHandler),
            (r"/info", InfoHandler),
            (r"/scheduler_info", SchedulerInfoHandler),
            (r"/status", StatusHandler),
            (r"/scale", ScaleHandler),
            (r"/adapt", AdaptHandler),
            (r"/register", RegisterPluginHandler)
        ]
    )


if __name__ == "__main__":
    app = make_app()
    app.listen(8788, "0.0.0.0")
    tornado.ioloop.IOLoop.current().start()
