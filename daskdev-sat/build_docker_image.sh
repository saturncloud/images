#

mkdir kubet
mkdir saturn


cp -r ../../saturn/pdc/kubet/ ./
cp  ../../saturn/pdc/dask/*.py  ./saturn/

docker build -t daskdev-sat .

rm -r ./kubet/
rm -r ./saturn/