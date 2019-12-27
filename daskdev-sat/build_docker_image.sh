#

mkdir -p saturn/pdc/dask
cp     ../../saturn/pdc/dask/*.py  ./saturn/pdc/dask/

pause
sudo docker build -t saturncloud/daskdev-sat .
#sudo docker push saturncloud/daskdev-sat

rm -r ./saturn/
