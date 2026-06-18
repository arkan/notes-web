---
title: No Results Test
---

# No Results Test

This page has a filter whose default value is absent from detected options, so
it should show zero rows on initial render.

```dataview
TABLE status, file.name as "Name"
FROM "Projects"
SORT file.name
FILTER status DEFAULT "nowhere"
```
