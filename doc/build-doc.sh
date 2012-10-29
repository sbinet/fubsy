#!/bin/sh -ex

rst2html \
  --title "Fubsy: User Guide" \
  --no-toc-backlinks \
  --section-numbering \
  --date --time \
  user-guide.rst > user-guide.html
