# images

This repository holds code to create Saturn Docker images.

## Adding a new image definition

Each image is stored in its own subdirectory. That subdirectory should have at least a `Dockerfile` and `.dockerignore`.

**`Dockerfile`**

A script that defines how to build the image.

For complete details on how to write `.dockerignore` files, see [the official docker documentation](https://docs.docker.com/engine/reference/builder/).

**`.dockerignore`**

Similar to `.gitignore`, `.dockerignore` is used to prevent unwanted files from being bundled in an image. For a good explanation of this, see ["Do Not Ignore .dockerignore"](https://codefresh.io/docker-tutorial/not-ignore-dockerignore-2/).

The images in this repository use `.dockerignore` files like this:

```text
*
!app.py
!environment.yml
```

That syntax says "ignore everything EXCEPT `app.py` and `environment.yml`".

For complete details on how to write `.dockerignore` files, see [the docker documentation](https://docs.docker.com/engine/reference/builder/#dockerignore-file).
