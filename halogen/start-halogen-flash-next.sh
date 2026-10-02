#!/bin/bash
#
export HSA_ENABLE_SDMA=0
export GPU_MAX_HW_QUEUES=8

#download the model: hf download peonist-ai/halogen-qwen3.8-flash-next --local-dir halogen-qwen3.8-flash-next
MODEL="/mnt/nas/huggingface/hub/halogen-qwen3.8-flash-next"

CONTAINER_NAME="halogen-qwen-3.8-flash-next"

docker stop $CONTAINER_NAME
sleep 2

docker run -d -t -p 1242:8731 --name $CONTAINER_NAME --rm \
  --device=/dev/kfd \
  --device=/dev/dri \
  --pid=host \
  --shm-size=32g \
  --ulimit memlock=-1:-1 \
  --memory=120g \
  --group-add video \
  --security-opt seccomp=unconfined \
  --security-opt label=type:container_runtime_t \
  --cap-add=sys_ptrace \
  -v ${MODEL}:/models \
  -v /opt/prompt-cache:/prompt-cache \
  -e HALOGEN_CTX=163840 \
  -e HALOGEN_MAX_TOK=32769 \
  -e HALOGEN_KV_POOL_POSITIONS=327680 \
  -e HALOGEN_KV_SLOTS=4 \
  -e HALOGEN_ADMIT_CHUNK=2 \
  -e HALOGEN_HOST_RESERVE_GIB=10 \
  -e HALOGEN_CACHE_DISK_GIB=768 \
  -e HALOGEN_PREFILL_KEEP_TRUNK=1 \
  -e HALOGEN_CACHE_DIR=/prompt-cache \
  ghcr.io/peonist-ai/halogen-flash-server:0.16.0

docker ps