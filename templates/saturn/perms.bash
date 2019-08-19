#!/bin/bash

set -ex

mkdir -p ${APP_BASE}
chown -R $NB_USER:$NB_USER ${APP_BASE}
