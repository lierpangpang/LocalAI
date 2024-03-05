#!/bin/bash
set -e
##
## A bash script installs the required dependencies of VALL-E-X and prepares the environment
export SHA=c0ddebaaaf8ffd1b3529c2bb654e650bce2f790f

SKIP_CONDA=${SKIP_CONDA:-0}

if [ $SKIP_CONDA -ne 1 ]; then
    source activate transformers
else
    export PATH=$PATH:/opt/conda/bin
    CONDA_PREFIX=$PWD
fi

echo $CONDA_PREFIX

git clone https://github.com/turboderp/exllamav2 $CONDA_PREFIX/exllamav2

pushd $CONDA_PREFIX/exllamav2

git checkout -b build $SHA

# TODO: this needs to be pinned within the conda environments
pip install -r requirements.txt

popd

cp -rfv $CONDA_PREFIX/exllamav2/* ./  

if [ "$PIP_CACHE_PURGE" = true ] ; then
    pip cache purge
fi