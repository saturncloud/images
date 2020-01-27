# Create main admin user
#
# This script sets up a database (migrate cmd) and then create one admin
# user for further interaction with the HTTP API in order to manage users
# and repos for the customer.

function gitea_migrate {
  /app/gitea/gitea migrate > /dev/null
}

function gitea_admin_create_user {
  /app/gitea/gitea admin create-user --username $ADMIN_USER --password $ADMIN_PASSWD --email "$ADMIN_EMAIL" --admin
}

RUN_ADMIN_OUT=$(gitea_migrate && gitea_admin_create_user | tail -n1)

if [[ $RUN_ADMIN_OUT == "New user '$ADMIN_USER' has been successfully created!" ]]; then
  echo $RUN_ADMIN_OUT
fi

# Since gitea CLI touches DB as root, we need to own it back
chown ${USER}:git /data/gitea.db
