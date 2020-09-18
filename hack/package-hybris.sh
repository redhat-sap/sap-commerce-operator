#!/bin/sh

SAP_COMMERCE_ZIP=$1

HYBRIS_PACKAGE_DIR="hybris-package"
HYBRIS_PACKAGE="hybris.tar.gz"

trap "rm -rf $HYBRIS_PACKAGE_DIR && rm -f $HYBRIS_PACKAGE" EXIT

echo "#####  Extracting hybris from SAP Commerce  #####"
unzip -q $SAP_COMMERCE_ZIP -d $HYBRIS_PACKAGE_DIR

echo "#####  Creating hybris package  #####"
cd $HYBRIS_PACKAGE_DIR && tar -czf ../$HYBRIS_PACKAGE hybris && cd ..
if [ $? -eq 0 ]; then
  echo "#####  Hybris package created  #####"
else
  exit
fi
