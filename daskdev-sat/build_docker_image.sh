#
#

mkdir kubet
mkdir saturn



cp -r ../../kubet/ ./
cp  ../*.py  ./saturn/
#cp -r ~/.kube  ./
#cp  ~/.minikube/ca.crt  . 
#cp  ~/.minikube/client.*  . 


docker build -t daskdev-sat .

rm -r ./kubet/
rm -r ./saturn/