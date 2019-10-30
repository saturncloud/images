FROM gitea/gitea:1.9.4

ARG ADMIN_USER=saturn
ENV ADMIN_USER $ADMIN_USER

ARG ADMIN_PASSWD=saturn
ENV ADMIN_PASSWD $ADMIN_PASSWD

ARG ADMIN_EMAIL=gitea@saturncloud.io
ENV ADMIN_EMAIL $ADMIN_EMAIL

COPY admin_create_user.sh /app/gitea/

RUN cat /app/gitea/admin_create_user.sh >> /etc/s6/gitea/setup