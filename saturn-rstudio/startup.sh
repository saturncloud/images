#!/bin/bash

set -e

/usr/lib/rstudio-server/bin/license-manager license-server $RSW_LICENSE_SERVER 
unset RSW_LICENSE_SERVER

echo "www-port=$RSW_PORT" >> /etc/rstudio/rserver.conf

# make a directory where rstudio-server has permission to place the pidfile
mkdir -p /var/run/rstudio
chown -R $USER:$USER /var/run/rstudio

# Deactivate license when it exists
deactivate() {
    echo "== Exiting =="

    echo "Deactivating license ..."
    sudo /usr/lib/rstudio-server/bin/license-manager deactivate >/dev/null 2>&1

    echo "== Done =="
}
trap deactivate EXIT

# touch log files to initialize them so the tail calls don't fail
mkdir -p /var/lib/rstudio-server
mkdir -p /var/lib/rstudio-launcher
mkdir -p /var/lib/rstudio-server/monitor
touch /var/lib/rstudio-server/monitor/log/rstudio-server.log
touch /var/lib/rstudio-launcher/rstudio-launcher.log
mkdir -p /var/lib/rstudio-launcher/Local
touch /var/lib/rstudio-launcher/Local/rstudio-local-launcher-placeholder.log
mkdir -p /var/lib/rstudio-launcher/Kubernetes
touch /var/lib/rstudio-launcher/Kubernetes/rstudio-kubernetes-launcher.log

update-locale LANG=en_US.utf8

tail -n 100 -f \
    /var/lib/rstudio-server/monitor/log/*.log \
    /var/lib/rstudio-launcher/*.log \
    /var/lib/rstudio-launcher/Local/*.log \
    /var/lib/rstudio-launcher/Kubernetes/*.log &

/usr/lib/rstudio-server/bin/rserver
