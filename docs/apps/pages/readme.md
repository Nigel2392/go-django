# Pages Application Reference Documentation

**Note**: This documentation page was (in part) generated by ChatGPT and has been fully reviewed by [Nigel2392](github.com/Nigel2392).

The Pages app provides a framework for managing page content with a hierarchical tree structure. It handles dynamic URL generation, registration of custom page types, and offers a suite of functions to operate on page nodes. This document covers the configuration, routing, registration, signals, and core operations of the Pages app.

---

## Table of Contents

1. [Overview](#overview)
2. [Configuration and Routing](#configuration)
3. [URL Handling](#routing-and-url-handling)
4. [Queries and Database Operations](./queries.md)
5. [Page Definitions and Registration](./contenttypes.md)
6. [PageNode Signals](./signals.md)
7. [`pages_models` Package Reference](./pages_models.md)
8. [Example Blog app](../../examples/blog.md)

---

## Overview

The Pages app is designed to manage page content similar to HTTP handlers, but with an underlying tree structure. Each page is represented by a node that holds the metadata (such as title, slug, and URL path) and hierarchical information (depth, parent, and children). Custom page types are registered using a `PageDefinition`, which allows the admin interface and routing system to work seamlessly with your content.

---

## Configuration

The Pages app is configured via a dedicated configuration object:

```go
type PageAppConfig struct {
    *apps.DBRequiredAppConfig
    backend            dj_models.Backend[models.Querier]
    routePrefix        string
    useRedirectHandler bool
}
```

- **NewAppConfig()**  
  Returns a new instance of `PageAppConfig` for initializing the Pages app.

- **App()**  
  Retrieves the current Pages app configuration.

- **QuerySet()**  
  Returns a `models.DBQuerier` to perform database operations related to page nodes.

You can easily retrieve the query set for database operations using the `QuerySet()` method.

Example:

```go
q := pages.App().QuerySet()
```

## Routing and URL Handling

The Pages app does not assume a default URL. You must explicitly define the route prefix and, if needed, enable a redirect handler.

- **SetRoutePrefix(prefix string)**  
  Sets the URL prefix from which the Pages app is served. All page URLs are prefixed with this value.

- **SetUseRedirectHandler(use bool)**  
  Configures whether to use a redirect handler for cases where only the page ID is known. This handler (registered at `/__pages__/redirect/<page_id>`) avoids an extra database lookup to determine the live URL.

- **URLPath(page Page) string**  
  Constructs and returns the live URL path for a given page. It combines the configured route prefix with the page node’s `UrlPath`.
