##
## A bash script wrapper that runs the diffusers server with conda

PATH=$PATH:/opt/conda/bin

# Activate conda environment
source activate vllm

# get the directory where the bash script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

python $DIR/backend_vllm.py
