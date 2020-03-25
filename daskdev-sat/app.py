import json

import tornado.ioloop
from tornado.web import RequestHandler, Application
from distributed import LocalCluster
from distributed.utils import ignoring


class SaturnCluster(LocalCluster):
    """Class that inherits from dask-kubernetes cluster

    NOTE: For now this is using LocalCluster for simplicity
    """
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
        self.cluster = SaturnCluster(dashboard_link='https://this.fake.url')


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


def make_saturn_cluster_app():
    return SaturnClusterApplication([
        (r"/", MainHandler),
        (r"/status", StatusHandler),
        (r"/scale", ScaleHandler),
        (r"/adapt", AdaptHandler),
    ])


if __name__ == "__main__":
    app = make_saturn_cluster_app()
    app.listen(8892)
    tornado.ioloop.IOLoop.current().start()
