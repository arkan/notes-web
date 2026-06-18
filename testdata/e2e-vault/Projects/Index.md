---
title: Project Index
---

# Project Index

This page lists all projects from the vault with filtering and sorting controls.

```dataview
TABLE status, tags, file.name as "Name", dateformat(file.mtime, "yyyy-MM-dd") as "Updated"
FROM "Projects"
SORT file.name
FILTER status DEFAULT "active" CLEARABLE
FILTER tags MODE multi
```

## Simple table

A basic table without filters.

```dataview
TABLE file.name as "Name", status
FROM "Projects"
SORT status
```
