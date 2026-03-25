# Projects API

The Projects endpoints allow you to manage the portfolio items displayed on the frontend. Results are served from an in-memory cache to ensure blazingly fast read performance.

## List All Projects

`GET /api/projects`

Returns all portfolio projects ordered by creation date (newest first).

**Response (200 OK)**

```json
{
  "success": true,
  "data": [
    {
      "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Smart Portfolio",
      "description": "A full-stack portfolio with AI chat",
      "tech_stack": "Go, PostgreSQL, pgvector",
      "github_url": "https://github.com/ZRishu/smart-portfolio",
      "live_url": "https://portfolio.example.com",
      "created_at": "2024-03-10T12:00:00Z"
    }
  ]
}
```

## Get Project by ID

`GET /api/projects/{id}`

Retrieves a single project by its UUID.

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "title": "Smart Portfolio",
    "description": "A full-stack portfolio with AI chat",
    "tech_stack": "Go, PostgreSQL, pgvector",
    "github_url": "https://github.com/ZRishu/smart-portfolio",
    "live_url": "https://portfolio.example.com",
    "created_at": "2024-03-10T12:00:00Z"
  }
}
```

## Create a Project

`POST /api/projects` *(Admin Protected)*

Creates a new project and invalidates the project cache.

**Request Body**

```json
{
  "title": "New Project",
  "description": "A description of the project",
  "tech_stack": "React, Node.js",
  "github_url": "https://github.com/username/project",
  "live_url": "https://project.com"
}
```

**Response (201 Created)**

Returns the created project object.

## Update a Project

`PUT /api/projects/{id}` *(Admin Protected)*

Updates an existing project. Replaces all mutable fields and invalidates the cache.

**Request Body**

```json
{
  "title": "Updated Project Title",
  "description": "Updated description",
  "tech_stack": "React, Node.js, Go",
  "github_url": "https://github.com/username/project",
  "live_url": "https://project.com"
}
```

**Response (200 OK)**

Returns the updated project object.

## Delete a Project

`DELETE /api/projects/{id}` *(Admin Protected)*

Permanently removes a project and invalidates the cache.

**Response (204 No Content)**
