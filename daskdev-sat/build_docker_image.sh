#


mkdir saturn
mkdir saturn/pdc
mkdir saturn/pdc/kube
mkdir saturn/pdc/dask


cp -r  ../../saturn/pdc/kubet/   ./saturn/pdc/kubet/
cp     ../../saturn/pdc/kube/*.py  ./saturn/pdc/kube/
cp     ../../saturn/pdc/dask/*.py  ./saturn/pdc/dask/
cp     ../../saturn/pdc/__init__.py  ./saturn/pdc/   # are we still using those?

docker build -t daskdev-sat .

rm -r ./saturn/
