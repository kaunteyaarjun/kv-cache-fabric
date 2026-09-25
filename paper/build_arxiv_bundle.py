#!/usr/bin/env python3
"""
build_arxiv_bundle.py
Packages paper/main.tex and paper/references.bib into an arXiv-ready .tar.gz and .zip archive.
"""

import os
import tarfile
import zipfile

def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    output_tar = os.path.join(script_dir, "kv_cache_fabric_arxiv.tar.gz")
    output_zip = os.path.join(script_dir, "kv_cache_fabric_arxiv.zip")

    files_to_bundle = [
        ("main.tex", os.path.join(script_dir, "main.tex")),
        ("references.bib", os.path.join(script_dir, "references.bib")),
    ]

    print("Building arXiv submission bundle...")

    # Build .tar.gz (arXiv standard submission format)
    with tarfile.open(output_tar, "w:gz") as tar:
        for arcname, filepath in files_to_bundle:
            if os.path.exists(filepath):
                tar.add(filepath, arcname=arcname)
                print(f"  Added {arcname} -> {output_tar}")
            else:
                print(f"  Error: {filepath} not found!")

    # Build .zip (Overleaf upload format)
    with zipfile.ZipFile(output_zip, "w", zipfile.ZIP_DEFLATED) as zipf:
        for arcname, filepath in files_to_bundle:
            if os.path.exists(filepath):
                zipf.write(filepath, arcname=arcname)
                print(f"  Added {arcname} -> {output_zip}")

    print(f"\nBundle generated successfully!")
    print(f"arXiv Tarball: {output_tar} ({os.path.getsize(output_tar)} bytes)")
    print(f"Overleaf Zip:  {output_zip} ({os.path.getsize(output_zip)} bytes)")

if __name__ == "__main__":
    main()
