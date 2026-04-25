# press

`press` is an opinionated static site generator for personal sites.

It builds a site from:

- a `content/` directory of Markdown posts with frontmatter
- a set of templ renderers
- a conventional blog structure

and produces a fully static HTML output.

---

## Quick Start

```go
err := press.Build(opts, siteData, renderers)
```

Where:

- `BuildOptions` defines filesystem and build behavior
- `SiteData` defines site-level metadata
- `Renderers` provide HTML output

---

## Model

press follows a staged build pipeline:

```
discover → parse → validate → sort → route → render → write → sync assets
```

Each stage has a clear responsibility:

- **discover**: find post files in `content/posts/`
- **parse**: split frontmatter and Markdown
- **validate**: enforce required fields and invariants
- **sort**: establish canonical ordering (newest first)
- **route**: assign URLs
- **render**: generate HTML via user-provided templates
- **write**: emit files to the output directory
- **sync assets**: copy static and per-post media

---

## Conventions

press is intentionally opinionated.

### Content structure

```
content/
  posts/
    my-post/
      index.md
      media/
```

### Frontmatter

Each post must define:

```yaml
---
title: My Post
slug: my-post
date: 2026-04-25
---
```

### Routes

```
/              → home
/blog/         → blog index
/blog/{slug}/  → post
```

### Assets

- Static assets are copied to `/assets/`
- Post-specific media is copied from `media/` directories

---

## Rendering

press does not own templates.

You provide renderers:

```go
type Renderers struct {
    Home      func(io.Writer, HomePageData) error
    BlogIndex func(io.Writer, BlogIndexPageData) error
    BlogPost  func(io.Writer, BlogPostPageData) error
}
```

Each renderer receives structured page data:

```go
type BlogPostPageData struct {
    Page PageData
    Post Post
}
```

Templates are responsible only for presentation.

---

## Data Model

### Post

```go
type Post struct {
    Slug  string
    URL   string
    Title string
    Date  time.Time
    Body  HTML
}
```

- `URL` is assigned during routing
- `Body` is precompiled HTML

### PageData

```go
type PageData struct {
    Site  SiteData
    Title string
}
```

### SiteData

```go
type SiteData struct {
    Title         string
    StylesheetURL string
}
```

---

## Philosophy

press treats content as structured data and HTML as a target format.

It favors:

- explicit structure over implicit behavior
- predictable conventions over configuration
- staged transformations over ad hoc rendering

The system is designed to be mechanically simple and easy to reason about.

---

## Status

press is currently a focused tool for personal sites.

Future work may include:

- post summaries / excerpts
- grouping (tags, dates)
- pagination

The current design is intended to support these without major structural changes.
