FROM saturncloud/saturnbase:0.9.3.6
USER root
RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
    r-base \
    r-base-core \
    r-base-core-dbg \
    r-base-dev \
    r-cran-abind \
    r-cran-acepack \
    r-cran-arm \
    r-cran-assertthat \
    r-cran-backports \
    r-cran-base64enc \
    r-cran-bbmisc \
    r-cran-bit \
    r-cran-bit64 \
    r-cran-bitops \
    r-cran-blob \
    r-cran-boot \
    r-cran-brew \
    r-cran-broom \
    r-cran-car \
    r-cran-cardata \
    r-cran-caret \
    r-cran-catools \
    r-cran-cellranger \
    r-cran-checkmate \
    r-cran-chron \
    r-cran-cli \
    r-cran-coda \
    r-cran-colorspace \
    r-cran-crayon \
    r-cran-curl \
    r-cran-data.table \
    r-cran-dbi \
    r-cran-dbplyr \
    r-cran-deldir \
    r-cran-desc \
    r-cran-devtools \
    r-cran-digest \
    r-cran-dplyr \
    r-cran-e1071 \
    r-cran-evaluate \
    r-cran-forcats \
    r-cran-formatr \
    r-cran-formula \
    r-cran-futile.logger \
    r-cran-futile.options \
    r-cran-ggplot2 \
    r-cran-git2r \
    r-cran-glmnet \
    r-cran-glue \
    r-cran-goftest \
    r-cran-gower \
    r-cran-gridextra \
    r-cran-gtable \
    r-cran-gtools \
    r-cran-haven \
    r-cran-highr \
    r-cran-hmisc \
    r-cran-hms \
    r-cran-htmltable \
    r-cran-htmltools \
    r-cran-htmlwidgets \
    r-cran-httr \
    r-cran-ipred \
    r-cran-isocodes \
    r-cran-jsonlite \
    r-cran-knitr \
    r-cran-labeling \
    r-cran-lambda.r \
    r-cran-latticeextra \
    r-cran-lava \
    r-cran-lazyeval \
    r-cran-lme4 \
    r-cran-lubridate \
    r-cran-magrittr \
    r-cran-maptools \
    r-cran-markdown \
    r-cran-matrixmodels \
    r-cran-memoise \
    r-cran-mime \
    r-cran-minqa \
    r-cran-modelmetrics \
    r-cran-munsell \
    r-cran-nloptr \
    r-cran-numderiv \
    r-cran-openssl \
    r-cran-pbkrtest \
    r-cran-pillar \
    r-cran-pkgconfig \
    r-cran-plogr \
    r-cran-plyr \
    r-cran-polyclip \
    r-cran-praise \
    r-cran-prettyunits \
    r-cran-prodlim \
    r-cran-progress \
    r-cran-proto \
    r-cran-purrr \
    r-cran-quantreg \
    r-cran-r6 \
    r-cran-randomforest \
    r-cran-rcolorbrewer \
    r-cran-rcpp \
    r-cran-rcppeigen \
    r-cran-rcpproll \
    r-cran-rcurl \
    r-cran-readr \
    r-cran-readxl \
    r-cran-recipes \
    r-cran-rematch \
    r-cran-reshape2 \
    r-cran-rjava \
    r-cran-rjson \
    r-cran-rlang \
    r-cran-rpostgresql \
    r-cran-rprojroot \
    r-cran-rsqlite \
    r-cran-rstudioapi \
    r-cran-scales \
    r-cran-snowballc \
    r-cran-sp \
    r-cran-sparsem \
    r-cran-spatstat \
    r-cran-spatstat.data \
    r-cran-spatstat.utils \
    r-cran-stringi \
    r-cran-stringr \
    r-cran-tensor \
    r-cran-testthat \
    r-cran-tibble \
    r-cran-tidyr \
    r-cran-tidyselect \
    r-cran-timedate \
    r-cran-utf8 \
    r-cran-viridis \
    r-cran-viridislite \
    r-cran-whisker \
    r-cran-withr \
    r-cran-xml2 \
    r-cran-xtable \
    r-cran-yaml \
    gfortran \
    psmisc \
    libapparmor1 \
    lsb-release \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
COPY install.R /tmp/install.R
RUN Rscript /tmp/install.R

RUN apt-get -qq update && \
    apt-get -qq install --yes --no-install-recommends \
    libzmq5 \
    libunwind8 \
    libunwind-dev \
    libzmq3-dev \
    libcurl4-openssl-dev \
    libssl-dev \
    > /dev/null && \
    apt-get -qq purge && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*
RUN Rscript -e "install.packages(c('repr', 'IRdisplay', 'IRkernel'), type = 'source')"
USER ${NB_USER}
RUN Rscript -e "IRkernel::installspec(prefix='/srv/conda')"