import os
from os.path import dirname, join, relpath
from jinja2 import Environment, FileSystemLoader
import subprocess
import logging

import click
import yaml
from sutils.files import ensure_directory

logger = logging.getLogger(__name__)
ROOT = dirname(__file__)
template_path = join(ROOT, "templates")
jinja_env = Environment(loader=FileSystemLoader([template_path]))
jinja_env.trim_blocks = True
jinja_env.lstrip_blocks = True


def template(data, template_path, dest, suffix):
    template_root = join(template_path, suffix)
    for root, dirs, files in os.walk(template_root):
        for f in files:
            path = join(root, f)
            tpath = relpath(path, template_path)
            rpath = relpath(path, template_root)
            _dest = join(dest, rpath)
            ensure_directory(_dest)
            with open(_dest, "w+") as f:
                print("writing to %s" % _dest)
                f.write(jinja_env.get_template(tpath).render(**data))


@click.group()
def cli():
    pass


def rsync(path1, path2, exclude_git=False):
    if path1.endswith("/"):
        path1 = path1[:-1]
    if path2.endswith("/"):
        path2 = path2[:-1]
    if exclude_git:
        cmd = "rsync -av --exclude .git/ %s/ %s"
    else:
        cmd = "rsync -av %s/ %s"
    try:
        cmd = cmd % (
            path1,
            path2,
        )
        _ = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    except subprocess.CalledProcessError as e:
        logger.error(e.output)
        raise


@cli.command()
@click.argument("config_path")
@click.option("--copy", is_flag=True)
def run(config_path, copy):
    with open(config_path) as f:
        data = yaml.load(f.read(), Loader=yaml.CLoader)
    out = join(dirname(__file__), "saturnbase")
    template(data, template_path, out, "saturnbase")
    out = join(dirname(__file__), "saturnbase-gpu")
    template(data, template_path, out, "saturnbase")
    out = join(dirname(__file__), "saturnbase-gpu")
    template(data, template_path, out, "saturnbase-gpu")
    out = join(dirname(__file__), "saturn")
    template(data, template_path, out, "saturn")
    if data["jsaturn_version"] == "local":
        if copy:
            rsync("../jupyterlab_saturn", join(out, "jupyterlab_saturn"), exclude_git=True)
    out = join(dirname(__file__), "saturn-gpu")
    template(data, template_path, out, "saturn-gpu")
    out = join(dirname(__file__), "scripts")
    template(data, template_path, out, "scripts")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    cli()
