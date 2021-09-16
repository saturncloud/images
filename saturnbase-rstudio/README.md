# RStudio Base image

This image lets you run RStudio in a docker container, so that eventually it will be an alternative to JupyterLab.

The docker container uses a central licensing server to get it's license. It has a special `saturn` user with password `saturn` to log in with (this can be changed via env vars).

Currently, we have a test licensing server running in an EC2 instance of our AWS account at the address `3.142.146.194`.

To run the container, use the following docker command:

```
docker run --privileged -t -d -p 8787:8787 -e RSW_LICENSE_SERVER=3.142.146.194 saturnbase-rstudio
```

To try it, then navigate to `localhost:8787` and log in with username `saturn` and password `saturn`.

## To do

1. Make it so when you go to the URL it drops you directly into RStudio without having to log in
2. Set this up for GPUs
3. Have the python installation in the correct location for use by Saturn Cloud setup
4. Set the user attributes correctly (name, UID) for Saturn
