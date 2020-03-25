import json
import os
import math
import yaml
import logging

import tornado.ioloop
from tornado.web import RequestHandler, Application

from distributed import LocalCluster
from distributed.utils import ignoring
from dask_kubernetes import KubeCluster, InCluster, make_pod_from_dict


logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)


NAME = os.environ.get("DASK_NAME")
NAMESPACE = os.environ.get("DASK_NAMESPACE", "main-namespace")
DASHBOARD_LINK = os.environ.get("DASK_DASHBOARD_LINK", None)
WORKER_CONFIG = "/etc/config/worker_spec.yaml"
TIMEOUT_MIN = os.environ.get("DASK_WORKER_TOUT", math.inf)


async def make_cluster():
    kube_auth = InCluster()  # with ServiceAccount
    await kube_auth.load()  # kubernetes authentication

    with open(WORKER_CONFIG) as f:
        d = yaml.safe_load(f)
        pod_template = make_pod_from_dict(d)

    return SaturnCluster(
        pod_template=pod_template,
        deploy_mode="remote",
        name=NAME,
        namespace=NAMESPACE,
        scheduler_timeout=f"{TIMEOUT_MIN} m",
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


class MainHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(str(cluster))


class StatusHandler(RequestHandler):
    def get(self):
        cluster = self.application.cluster
        self.write(cluster.status)


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
        (r"/status", StatusHandler),
        (r"/scale", ScaleHandler),
        (r"/adapt", AdaptHandler),
    ])


if __name__ == "__main__":
    app = make_app()
    app.listen(80)
    tornado.ioloop.IOLoop.current().start()
