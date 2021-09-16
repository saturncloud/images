#!/bin/bash

set -e
set -x

/usr/lib/rstudio-server/bin/license-manager license-server $RSW_LICENSE_SERVER
unset RSW_LICENSE_SERVER

echo "www-port=$RSW_PORT" >> rserver.conf

# Deactivate license when it exists
deactivate() {
    echo "== Exiting =="
    rstudio-server stop

    echo " --> TAIL 100 rstudio-server.log"
    tail -n 100 /var/log/rstudio-server.log
    echo " --> TAIL 100 rstudio-kubernetes-launcher.log"
    tail -n 100 /var/lib/rstudio-launcher/Kubernetes/rstudio-kubernetes-launcher.log
    echo " --> TAIL 100 rstudio-local-launcher*.log"
    tail -n 100 /var/lib/rstudio-launcher/Local/rstudio-local-launcher*.log
    echo " --> TAIL 100 rstudio-launcher.log"
    tail -n 100 /var/lib/rstudio-launcher/rstudio-launcher.log
    echo " --> TAIL 100 monitor/log/rstudio-server.log"
    tail -n 100 /var/lib/rstudio-server/monitor/log/rstudio-server.log

    echo "Deactivating license ..."
    /usr/lib/rstudio-server/bin/license-manager deactivate >/dev/null 2>&1

    echo "== Done =="
}
trap deactivate EXIT

# touch log files to initialize them
su rstudio-server -c 'touch /var/lib/rstudio-server/monitor/log/rstudio-server.log'
mkdir -p /var/lib/rstudio-launcher
chown rstudio-server:rstudio-server /var/lib/rstudio-launcher
su rstudio-server -c 'touch /var/lib/rstudio-launcher/rstudio-launcher.log'
touch /var/log/rstudio-server.log
mkdir -p /var/lib/rstudio-launcher/Local
chown rstudio-server:rstudio-server /var/lib/rstudio-launcher/Local
su rstudio-server -c 'touch /var/lib/rstudio-launcher/Local/rstudio-local-launcher-placeholder.log'
mkdir -p /var/lib/rstudio-launcher/Kubernetes
chown rstudio-server:rstudio-server /var/lib/rstudio-launcher/Kubernetes
su rstudio-server -c 'touch /var/lib/rstudio-launcher/Kubernetes/rstudio-kubernetes-launcher.log'

# Create one user
if [ $(getent passwd $RSW_USER_UID) ] ; then
    echo "UID $RSW_USER_UID already exists, not creating $RSW_USER test user";
else
    if [ -z "$RSW_USER" ]; then
        echo "Empty 'RSW_USER' variables, not creating test user";
    else
        useradd -m -s /bin/bash -N -u $RSW_USER_UID $RSW_USER
        echo "$RSW_USER:$RSW_USER_PASSWD" | sudo chpasswd
    fi
fi

tail -n 100 -f \
  /var/lib/rstudio-server/monitor/log/*.log \
  /var/lib/rstudio-launcher/*.log \
  /var/lib/rstudio-launcher/Local/*.log \
  /var/lib/rstudio-launcher/Kubernetes/*.log \
  /var/log/rstudio-launcher.log \
  /var/log/rstudio-server.log &

# the main container process
# cannot use "exec" or the "trap" will be lost
/usr/lib/rstudio-server/bin/rserver --server-daemonize 0 > /var/log/rstudio-server.log 2>&1
