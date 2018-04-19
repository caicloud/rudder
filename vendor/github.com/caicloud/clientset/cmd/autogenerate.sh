# Copyright 2017 caicloud authors. All rights reserved.

# The script auto-generate kubernetes clients, listers, informers

set -e

ORIGIN=$(pwd)
cd $(dirname ${BASH_SOURCE[0]})

# Add your packages here
PKGS=$(find ../pkg/apis/ -type d -mindepth 2 -maxdepth 2 2>/dev/null | sed 's|^\.\./pkg/apis/*||g')

echo $PKGS

CLIENT_PATH=github.com/caicloud/clientset
CLIENT_APIS=$CLIENT_PATH/pkg/apis

for path in $PKGS
do
	ALL_PKGS="$CLIENT_APIS/$path "$ALL_PKGS
done

function join {
	local IFS="$1"
   	shift
   	result="$*"
}

join "," ${PKGS[@]}
PKGS=$result

join "," $ALL_PKGS
FULL_PKGS=$result

echo "PKGS: $PKGS"
echo "FULL PKGS: $FULL_PKGS"


BINS=(
  client-gen
  conversion-gen
  deepcopy-gen
  defaulter-gen
  informer-gen
  lister-gen
)

TEMPBIN=./tmpbin

mkdir -p $TEMPBIN
for bin in "${BINS[@]}"
do
	go build -o $TEMPBIN/$bin ./$bin
done

$TEMPBIN/conversion-gen \
  -i $FULL_PKGS \
  -O "zz_generated.conversion"

echo "Generated conversion"


$TEMPBIN/defaulter-gen \
  -i $FULL_PKGS \
  -O "zz_generated.defaults"

echo "Generated defaulter"

$TEMPBIN/deepcopy-gen \
  -i $FULL_PKGS \
  -O "zz_generated.deepcopy"

echo "Generated deepcopy"

$TEMPBIN/client-gen \
  -n kubernetes \
  --clientset-path $CLIENT_PATH \
  --input-base $CLIENT_APIS \
  --input $PKGS

echo "Generated clients"

$TEMPBIN/lister-gen \
  -p $CLIENT_PATH/listers \
  -i $FULL_PKGS

echo "Generated listers"

$TEMPBIN/informer-gen \
  --single-directory \
  -p $CLIENT_PATH/informers \
  --versioned-clientset-package $CLIENT_PATH/kubernetes \
  --listers-package $CLIENT_PATH/listers \
  -i $FULL_PKGS

echo "Generated informers"


rm -rf $TEMPBIN

cd $ORIGIN
