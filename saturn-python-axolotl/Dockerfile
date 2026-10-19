ARG SATURNBASE_GPU_IMAGE
FROM ${SATURNBASE_GPU_IMAGE}

RUN sudo apt-get -qq --allow-releaseinfo-change update && \
    sudo apt-get -qq install --yes --no-install-recommends \
    libgl1

# Blackwell Ultra (B300 / GB300, compute capability 10.3, sm_103a): Triton 3.5
# (torch 2.9) bundles a CUDA 12.8 ptxas that rejects sm_103a, which killed every
# LoRA fine-tune at its first step (axolotl's LoRA Triton kernels). Triton 3.6
# (torch 2.10, pinned in environment.yml) uses its own CUDA 12.9 ptxas-blackwell
# for 10.x and works without this; pointing TRITON_PTXAS_PATH at the devel
# base's CUDA 13.x ptxas keeps a Triton downgrade from bringing the failure back.
# Both proven on a B300, 2026-10-18.
ENV TRITON_PTXAS_PATH=/usr/local/cuda/bin/ptxas
# Blackwell Ultra (compute capability 10.3), torch 2.10.0+cu130 / cuBLAS 13.1: every F.linear WITH a bias
# fails with CUBLAS_STATUS_NOT_INITIALIZED in cublasLtMatmulAlgoGetHeuristic (PyTorch's cuBLASLt
# bias-epilogue addmm path; bf16, fp16 and fp32 alike). Fine-tunes hit it whenever axolotl's fused LoRA
# kernels are off (lora_dropout > 0, which Atlas sends), e.g. Qwen2's biased q/k/v. DISABLE_ADDMM_CUDA_LT=1
# makes PyTorch use plain cuBLAS GEMM plus a bias add for addmm (negligible cost for LoRA training).
# Proven on a B300 (x86) with Atlas's hyperparameters, 2026-10-19.
ENV DISABLE_ADDMM_CUDA_LT=1

COPY environment.yml /tmp/environment.yml
# Unlike the serving image (saturn-python-llm), axolotl is installed WITH its
# deps here: this image has no vLLM, so there is no transformers<5 constraint to
# protect, and axolotl 0.16.1 needs the transformers 5.x API at runtime. Letting
# it resolve its own tree (transformers 5.5, datasets 4.5, trl 0.29, hf-hub>=1,
# accelerate 1.13, ...) is what makes training actually work. axolotl is declared
# in environment.yml's pip: block with its extras, so a normal env update pulls
# everything; no separate --no-deps step.
RUN mamba env update -n saturn --file /tmp/environment.yml && \
    ${CONDA_DIR}/envs/saturn/bin/python -m ipykernel install \
        --name python3 \
        --display-name 'saturn (Python 3)' \
        --prefix=${CONDA_DIR} && \
    ${CONDA_DIR}/bin/conda clean -afy && \
    find ${CONDA_DIR} -type f,l -name '*.pyc' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.a' -delete && \
    find ${CONDA_DIR} -type f,l -name '*.js.map' -delete
RUN echo '' > ${CONDA_DIR}/envs/saturn/conda-meta/history
