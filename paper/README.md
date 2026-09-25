# Disaggregated KV-Cache Fabric: arXiv Submission & LaTeX Package

This directory contains the publication-ready LaTeX paper and bibliography ready for submission to **arXiv**, **Overleaf**, or systems conferences (EuroSys, OSDI, SOSP, USENIX ATC).

---

## Files

- **`main.tex`**: The complete, self-contained two-column LaTeX paper with all mathematical models, concurrency invariants, algorithms, listings, control-plane tables, and empirical SmolLM2-1.7B telemetry.
- **`references.bib`**: BibTeX bibliography containing full citations for PagedAttention, SGLang, DistServe, Mooncake, FlashAttention, Sarathi, Splitwise, and SmolLM2.
- **`kv_cache_fabric_arxiv.tar.gz`**: Standard arXiv submission tarball containing `main.tex` and `references.bib`.
- **`kv_cache_fabric_arxiv.zip`**: Ready-to-import Overleaf ZIP archive.
- **`build_arxiv_bundle.py`**: Automated script to repackage the `.tar.gz` and `.zip` bundles.

---

## 1. Importing to Overleaf (Recommended)

1. Open [Overleaf](https://www.overleaf.com/).
2. Click **New Project** $\to$ **Upload Project**.
3. Select `paper/kv_cache_fabric_arxiv.zip` (or drag and drop it into Overleaf).
4. Overleaf will automatically detect `main.tex` and compile the PDF preview.

---

## 2. Submitting to arXiv

1. Go to [arXiv Submission Portal](https://arxiv.org/submit).
2. Start a new submission and select primary category:
   - **`cs.DC` (Distributed, Parallel, and Cluster Computing)** or
   - **`cs.OS` (Operating Systems)** / **`cs.AI` (Artificial Intelligence)**.
3. In the **Upload Files** step, upload `paper/kv_cache_fabric_arxiv.tar.gz`.
4. The arXiv Auto-TeX compiler will process `main.tex` and `references.bib` and render the official arXiv preprint PDF.

---

## 3. Compiling Locally with TeX Live / MiKTeX

If you have TeX Live or MiKTeX installed:

```bash
cd paper
pdflatex main.tex
bibtex main
pdflatex main.tex
pdflatex main.tex
```

This will produce `main.pdf`.
