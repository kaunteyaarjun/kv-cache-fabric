# Disaggregated KV-Cache Fabric: LaTeX Paper & Preprint Submission Package

This directory contains the publication-ready LaTeX paper and bibliography ready for submission to **Zenodo**, **Overleaf**, or systems conferences (EuroSys, OSDI, SOSP, USENIX ATC).

> 📌 **Looking to cite or reference?** See the Zenodo record details: [`docs/PREPRINT_SUBMISSION_GUIDE.md`](../docs/PREPRINT_SUBMISSION_GUIDE.md)

---

## Files

- **`main.tex`**: The complete, self-contained two-column LaTeX paper with all mathematical models, concurrency invariants, algorithms, listings, control-plane tables, and empirical SmolLM2-1.7B telemetry.
- **`references.bib`**: BibTeX bibliography containing full citations for PagedAttention, SGLang, DistServe, Mooncake, FlashAttention, Sarathi, Splitwise, and SmolLM2.

---

## 1. Importing to Overleaf

1. Open [Overleaf](https://www.overleaf.com/).
2. Create a **New Project** $\to$ **Blank Project**.
3. Upload `main.tex` and `references.bib`.
4. Overleaf will automatically compile and render the preprint PDF.

---

## 2. Zenodo DOI Record

See [`docs/PREPRINT_SUBMISSION_GUIDE.md`](../docs/PREPRINT_SUBMISSION_GUIDE.md) for the official published record and BibTeX citation (`10.5281/zenodo.22966805`).

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
