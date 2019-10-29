#


mkdir saturn
mkdir saturn/pdc
mkdir saturn/pdc/kube
mkdir saturn/pdc/dask


cp -r  ../../saturn/pdc/kubet/   ./saturn/pdc/kubet/
cp     ../../saturn/pdc/kube/*.py  ./saturn/pdc/kube/
cp     ../../saturn/pdc/dask/*.py  ./saturn/pdc/dask/


docker build -t daskdev-sat .

#rm -r ./kubet/
#rm -r ./saturn/
#rm -r ./kube/