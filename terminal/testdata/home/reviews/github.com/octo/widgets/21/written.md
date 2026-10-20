---
stack: 21, 22
head: cd3c70277eb3b388699274ec861c66932edb9348
title: Search, then a basket
---
## What this changes

#21 adds a search over the catalog in `Search.swift:4`. #22 gives the basket a search of its own, in `Basket.swift:7`. [Likely]

## Risks

`MemoryCatalog.swift:9` appends without checking for duplicates. [Guessing]
