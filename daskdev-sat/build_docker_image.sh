#

mkdir saturn
mkdir saturn/pdc
mkdir saturn/pdc/kube
mkdir saturn/pdc/dask


cp -r  ../../saturn/pdc/kubet/   ./saturn/pdc/kubet/
cp     ../../saturn/pdc/dask/*.py  ./saturn/pdc/dask/


sudo docker build -t saturncloud/daskdev-sat .
#sudo docker push saturncloud/daskdev-sat

rm -r ./saturn/
