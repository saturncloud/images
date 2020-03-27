import json
import os
import yaml
import logging

import tornado.ioloop
from tornado.web import RequestHandler, Application

from distributed.utils import ignoring
from dask_kubernetes import KubeCluster, make_pod_from_dict


logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)


NAME = os.environ.get("DASK_NAME")
NAMESPACE = os.environ.get("DASK_NAMESPACE", "main-namespace")
DASHBOARD_LINK = os.environ.get("DASK_DASHBOARD_LINK", None)
WORKER_CONFIG = "/etc/config/worker_spec.yaml"


def make_cluster():
    with open(WORKER_CONFIG) as f:
        d = yaml.safe_load(f)
        pod_template = make_pod_from_dict(d)

    log.info(f"Starting dask cluster {NAME} in namespace {NAMESPACE}")
    log.info(f"worker config: {WORKER_CONFIG}")
    log.info(f"dashboard address: {DASHBOARD_LINK}")

    return SaturnCluster(
        pod_template=pod_template,
        deploy_mode="remote",
        name=NAME,
        namespace=NAMESPACE,
        dashboard_link=DASHBOARD_LINK
    )


class SaturnCluster(KubeCluster):
    """Class that inherits from dask-kubernetes cluster"""
    def __init__(self, *args, dashboard_link=None, **kwargs):
        """Init as usual, but add dashboard_link to object"""
        super().__init__(*args, **kwargs)
        self._dashboard_link = dashboard_link

    @property
    def dashboard_link(self):
        """Overwrite base class and just return attr"""
        return self._dashboard_link


class SaturnClusterApplication(Application):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.cluster = make_cluster()
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
            "dashboard_link": cluster.dashboard_link
        }
        self.write(json.dumps(info))


class SchedulerInfoHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(json.dumps(cluster.scheduler_info))


class StatusHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(json.dumps({"status": cluster.status}))


class ScaleHandler(RequestHandler):
    def post(self):
        cluster = self.application.cluster
        with ignoring(AttributeError):
            cluster._adaptive.stop()
        body = json.loads(self.request.body)
        cluster.scale(**body)


class AdaptHandler(RequestHandler):
    def post(self):
        cluster = self.application.cluster
        body = json.loads(self.request.body)
        cluster.adapt(**body)


def make_app():
    return SaturnClusterApplication([
        (r"/", MainHandler),
        (r"/info", InfoHandler),
        (r"/scheduler_info", SchedulerInfoHandler),
        (r"/status", StatusHandler),
        (r"/scale", ScaleHandler),
        (r"/adapt", AdaptHandler),
    ])


if __name__ == "__main__":
    app = make_app()
    app.listen(8788, "0.0.0.0")
    tornado.ioloop.IOLoop.current().start()
