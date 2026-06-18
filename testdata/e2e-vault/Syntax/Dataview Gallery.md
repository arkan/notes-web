---
title: Dataview Gallery
tags: [demo, dataview]
---

# Dataview Gallery

A page exercising all supported Dataview query types.

## TABLE

```dataview
TABLE file.name as "Name", file.folder as "Folder"
FROM "Syntax"
SORT file.name
```

## LIST

```dataview
LIST
FROM "Syntax"
SORT file.name
```

## TASK

```dataview
TASK
FROM "Tasks"
```

## CALENDAR

```dataview
CALENDAR file.mtime
FROM "Syntax"
```

## GROUP BY

```dataview
TABLE file.name as "Name"
FROM "Syntax"
GROUP BY file.folder
SORT file.name
```

## FLATTEN

```dataview
TABLE file.name as "Name", file.tags as "Tags"
FROM "Syntax"
FLATTEN file.tags
SORT file.name
```

## WHERE + LIMIT

```dataview
TABLE file.name as "Name"
FROM "Syntax"
WHERE contains(file.tags, "demo")
LIMIT 5
SORT file.name
```

## Unsupported block (diagnostics)

```dataview
CALENDAR gibberish-without-date
```

```dataviewjs
// deliberately unsupported
console.log("dataviewjs blocks are not executed")
```
