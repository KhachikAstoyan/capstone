# API Documentation

Base URL: `/api/v1`

## Table of Contents

- [Authentication](#authentication)
- [Auth Endpoints](#auth-endpoints)
- [Problems](#problems)
- [Tags](#tags)
- [Roles](#roles)
- [Permissions](#permissions)
- [Users](#users)
- [Error Responses](#error-responses)

---

## Authentication

Protected endpoints require a JWT access token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

Some endpoints use an HttpOnly cookie (`refresh_token`) instead of the header — these are noted explicitly.

### Permission Levels

| Permission key            | Description                                         |
| ------------------------- | --------------------------------------------------- |
| `admin.access`            | General admin panel access                          |
| `problems.manage`         | Create, update, delete problems and tags            |
| `rbac.roles.view`         | View roles                                          |
| `rbac.roles.manage`       | Create, update, delete roles and assign permissions |
| `rbac.permissions.view`   | View permissions                                    |
| `rbac.permissions.manage` | Create, update, delete permissions                  |
| `rbac.users.view`         | View user roles and permissions                     |
| `rbac.users.manage`       | Assign and remove roles from users                  |

---

## Common Models

### `User`

```json
{
  "id": "uuid",
  "handle": "string",
  "email": "string",
  "email_verified": false,
  "display_name": "string",
  "avatar_url": "string",
  "status": "ACTIVE | BANNED",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
```

### `PublicUserProfile`

Returned by [Get public user profile](#get-public-user-profile). Omits email and account flags suitable for any visitor.

```json
{
  "id": "uuid",
  "handle": "johndoe",
  "display_name": "John Doe",
  "avatar_url": "https://…",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### `Problem`

```json
{
  "id": "uuid",
  "slug": "two-sum",
  "title": "Two Sum",
  "summary": "Given an array of integers, return indices of two numbers that add up to the target.",
  "statement_markdown": "<p>Given an array...</p>",
  "time_limit_ms": 1000,
  "memory_limit_mb": 256,
  "tests_ref": "s3://bucket/tests/two-sum",
  "tests_hash": "string",
  "visibility": "draft | published | archived",
  "difficulty": "easy | medium | hard",
  "created_by_user_id": "uuid",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "tags": ["array", "hash-table"],
  "acceptance_rate": 45.6,
  "is_solved": false
}
```

> `acceptance_rate` is computed dynamically as `(accepted submissions / total submissions) * 100`, counting only `submission` type (not `test_run`).  
> `is_solved` is `true` if the authenticated user has an accepted submission for this problem.  
> `tags`, `acceptance_rate`, and `is_solved` may be omitted when not applicable.

### `Tag`

```json
{
  "id": "uuid",
  "name": "array",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### `Role`

```json
{
  "id": "uuid",
  "name": "admin",
  "description": "string",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### `Permission`

```json
{
  "id": "uuid",
  "key": "problems.manage",
  "description": "string",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

## Auth Endpoints

### Register

**`POST /auth/register`**

Creates a new user account and queues a verification email. Does **not** create a session; the client must not treat the user as logged in until they verify their email and call [Login](#login).

**Request body:**

```json
{
  "handle": "johndoe",
  "email": "john@example.com",
  "password": "secret123",
  "display_name": "John Doe"
}
```

| Field          | Type   | Required | Description                           |
| -------------- | ------ | -------- | ------------------------------------- |
| `handle`       | string | Yes      | Unique username                       |
| `email`        | string | Yes      | Unique email address                  |
| `password`     | string | Yes      | Must meet minimum length requirements |
| `display_name` | string | No       | Display name                          |

**Response `201`:**

```json
{
  "message": "Registration successful. Check your email to verify your account before signing in."
}
```

No `refresh_token` cookie is set.

**Errors:** `400` invalid input, `409` email or handle already taken.

---

### Login

**`POST /auth/login`**

**Request body:**

```json
{
  "email": "john@example.com",
  "password": "secret123"
}
```

**Response `200`:** Access token, refresh token, expiry, and `user` (same shape as a successful refresh). Sets an HttpOnly `refresh_token` cookie.

**Errors:** `401` invalid credentials, `403` user is banned or email not verified yet.

---

### Refresh Token

**`POST /auth/refresh`**

Exchanges a valid refresh token for a new token pair. Does not accept a JSON body — reads the `refresh_token` HttpOnly cookie.

**Response `200`:** Same as [Register](#register) with a new cookie.

**Errors:** `401` missing, invalid, or expired refresh token, `403` user is banned.

---

### Logout

**`POST /auth/logout`**

Revokes the refresh token and clears the cookie. Reads the `refresh_token` HttpOnly cookie.

**Response `200`:**

```json
{ "message": "logged out successfully" }
```

---

### Verify Email

**`POST /auth/verify-email`**

Consumes a one-time email verification token (same value emailed to the user after registration) and sets `email_verified` on the account.

**Request body:**

```json
{
  "token": "base64url-encoded-secret-from-email"
}
```

| Field   | Type   | Required | Description                          |
| ------- | ------ | -------- | ------------------------------------ |
| `token` | string | Yes      | Plain verification secret from link |

**Response `200` (verified):**

```json
{
  "status": "verified",
  "message": "email verified successfully"
}
```

**Response `200` (link expired on server — a new token is stored and an email should be sent when outbound mail is wired):**

```json
{
  "status": "resent",
  "message": "this verification link expired; a new verification email will be sent"
}
```

**Errors:** `400` missing/invalid token or unknown/already-used token, `500` server error.

Clients should open the app with query param `?token=<secret>` from the email link; the SPA calls this endpoint and then removes `token` from the URL.

---

### Get Current User

**`GET /auth/me`** — _Requires auth_

**Response `200`:** [`User`](#user) object.

**Errors:** `401` missing/invalid token, `404` user not found.

---

### Get My Roles

**`GET /auth/me/roles`** — _Requires auth_

**Response `200`:**

```json
[{ ...Role }]
```

---

### Get My Permissions

**`GET /auth/me/permissions`** — _Requires auth_

**Response `200`:**

```json
[{ ...Permission }]
```

---

## Problems

### List Problems

**`GET /problems/`**

Returns a paginated list of problems. Optionally enriches results with `is_solved` and `acceptance_rate` when the user is authenticated.

**Query parameters:**

| Parameter    | Type                | Default | Description                                            |
| ------------ | ------------------- | ------- | ------------------------------------------------------ |
| `limit`      | integer             | `50`    | Number of results to return                            |
| `offset`     | integer             | `0`     | Number of results to skip                              |
| `visibility` | string              | —       | Filter by `draft`, `published`, or `archived`          |
| `difficulty` | string              | —       | Filter by `easy`, `medium`, or `hard`                  |
| `search`     | string              | —       | Case-insensitive search on title, slug, and summary      |
| `tags[]`     | string (repeatable) | —       | Filter by tag names, e.g. `tags[]=array&tags[]=string` |

**Response `200`:**

```json
{
  "problems": [{ ...Problem }],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

---

### Get Problem by ID

**`GET /problems/{id}`**

**Path parameters:** `id` — problem UUID.

**Response `200`:** [`Problem`](#problem) object.

**Errors:** `404` not found.

---

### Get Problem by Slug

**`GET /problems/slug/{slug}`**

**Path parameters:** `slug` — problem slug string.

**Response `200`:** [`Problem`](#problem) object.

**Errors:** `404` not found.

---

### Get Problem Tags

**`GET /problems/{id}/tags`**

**Path parameters:** `id` — problem UUID.

**Response `200`:**

```json
{
  "tags": [{ ...Tag }]
}
```

Returns an empty array if the problem has no tags.

---

### Create Problem _(Admin)_

**`POST /internal/problems/`** — _Requires auth + `admin.access` + `problems.manage`_

The slug is auto-generated from the title (with a numeric suffix on conflict).

**Request body:**

```json
{
  "title": "Two Sum",
  "summary": "Plain-text blurb for list views (optional, max 500 chars).",
  "statement_markdown": "<p>Given an array of integers...</p>",
  "time_limit_ms": 1000,
  "memory_limit_mb": 256,
  "tests_ref": "s3://bucket/tests/two-sum",
  "visibility": "draft",
  "difficulty": "easy"
}
```

| Field                | Type    | Required | Description                         |
| -------------------- | ------- | -------- | ----------------------------------- |
| `title`              | string  | Yes      | Problem title                       |
| `summary`            | string  | No       | Plain-text list blurb (max 500 Unicode scalars); not markdown |
| `statement_markdown` | string  | Yes      | Problem statement (HTML supported)  |
| `time_limit_ms`      | integer | Yes      | Time limit in milliseconds          |
| `memory_limit_mb`    | integer | Yes      | Memory limit in megabytes           |
| `tests_ref`          | string  | Yes      | Path/URI to the test case bundle    |
| `visibility`         | string  | Yes      | `draft`, `published`, or `archived` |
| `difficulty`         | string  | Yes      | `easy`, `medium`, or `hard`         |

**Response `201`:** [`Problem`](#problem) object.

---

### Update Problem _(Admin)_

**`PUT /internal/problems/{id}`** — _Requires auth + `admin.access` + `problems.manage`_

All fields are optional — only provided fields are updated.

**Request body:**

```json
{
  "slug": "two-sum",
  "title": "Two Sum",
  "summary": "Updated plain-text summary for the list.",
  "statement_markdown": "<p>Given an array of integers...</p>",
  "time_limit_ms": 1000,
  "memory_limit_mb": 256,
  "tests_ref": "s3://bucket/tests/two-sum",
  "visibility": "published",
  "difficulty": "medium"
}
```

Optional body fields include `summary` (plain text, max 500 Unicode scalars).

**Response `200`:** [`Problem`](#problem) object.

**Errors:** `404` not found.

---

### Delete Problem _(Admin)_

**`DELETE /internal/problems/{id}`** — _Requires auth + `admin.access` + `problems.manage`_

**Response `200`:**

```json
{ "message": "problem deleted successfully" }
```

**Errors:** `404` not found.

---

### Update Problem Tags _(Admin)_

**`PUT /internal/problems/{id}/tags`** — _Requires auth + `admin.access` + `problems.manage`_

Replaces the full set of tags for a problem. Pass an empty array to clear all tags.

**Request body:**

```json
{
  "tag_ids": ["uuid1", "uuid2"]
}
```

**Response `200`:**

```json
{ "message": "tags updated successfully" }
```

---

## Tags

### List Tags

**`GET /tags/`**

Returns all available tags.

**Response `200`:**

```json
{
  "tags": [{ ...Tag }]
}
```

Returns an empty array if no tags exist.

---

### Create Tag _(Admin)_

**`POST /internal/tags/`** — _Requires auth + `admin.access` + `problems.manage`_

Idempotent — returns the existing tag if a tag with the same name already exists.

**Request body:**

```json
{
  "name": "array"
}
```

**Response `201`:** [`Tag`](#tag) object.

---

## Roles

All role endpoints require **auth + `rbac.roles.view`** (read) or **`rbac.roles.manage`** (write).

### List Roles

**`GET /roles/`** — _Requires `rbac.roles.view`_

**Response `200`:** `[]Role`

---

### Get Role

**`GET /roles/{roleID}`** — _Requires `rbac.roles.view`_

**Response `200`:** [`Role`](#role) object.

---

### Get Role with Permissions

**`GET /roles/{roleID}/permissions`** — _Requires `rbac.roles.view`_

**Response `200`:**

```json
{
  "role": { ...Role },
  "permissions": [{ ...Permission }]
}
```

---

### Create Role

**`POST /roles/`** — _Requires `rbac.roles.manage`_

**Request body:**

```json
{
  "name": "moderator",
  "description": "Can manage content"
}
```

**Response `201`:** [`Role`](#role) object.

---

### Update Role

**`PUT /roles/{roleID}`** — _Requires `rbac.roles.manage`_

**Request body:**

```json
{
  "name": "moderator",
  "description": "Updated description"
}
```

**Response `200`:** [`Role`](#role) object.

---

### Delete Role

**`DELETE /roles/{roleID}`** — _Requires `rbac.roles.manage`_

**Response `200`:**

```json
{ "message": "role deleted successfully" }
```

---

### Assign Permission to Role

**`POST /roles/{roleID}/permissions`** — _Requires `rbac.roles.manage`_

**Request body:**

```json
{
  "permission_id": "uuid"
}
```

**Response `200`:**

```json
{ "message": "permission assigned successfully" }
```

---

### Remove Permission from Role

**`DELETE /roles/{roleID}/permissions/{permissionID}`** — _Requires `rbac.roles.manage`_

**Response `200`:**

```json
{ "message": "permission removed successfully" }
```

---

## Permissions

All permission endpoints require **auth + `rbac.permissions.view`** (read) or **`rbac.permissions.manage`** (write).

### List Permissions

**`GET /permissions/`** — _Requires `rbac.permissions.view`_

**Response `200`:** `[]Permission`

---

### Get Permission

**`GET /permissions/{permissionID}`** — _Requires `rbac.permissions.view`_

**Response `200`:** [`Permission`](#permission) object.

---

### Create Permission

**`POST /permissions/`** — _Requires `rbac.permissions.manage`_

**Request body:**

```json
{
  "key": "problems.manage",
  "description": "Manage problems and tags"
}
```

**Response `201`:** [`Permission`](#permission) object.

---

### Update Permission

**`PUT /permissions/{permissionID}`** — _Requires `rbac.permissions.manage`_

**Request body:**

```json
{
  "key": "problems.manage",
  "description": "Updated description"
}
```

**Response `200`:** [`Permission`](#permission) object.

---

### Delete Permission

**`DELETE /permissions/{permissionID}`** — _Requires `rbac.permissions.manage`_

**Response `200`:**

```json
{ "message": "permission deleted successfully" }
```

---

## Users

### Get public user profile

**`GET /users/{userRef}`** — _Public (no auth)_

Loads a user by **UUID** (`userRef` parses as a UUID) or by **handle** (case-sensitive, unique). Banned accounts respond as not found.

**Path parameters:** `userRef` — user id or handle.

**Response `200`:** [`PublicUserProfile`](#publicuserprofile) object.

**Errors:** `400` empty or invalid reference, `404` user not found or banned.

---

### Get User Roles

**`GET /users/{userID}/roles`** — _Requires auth + `rbac.users.view`_

**Response `200`:** `[]Role`

---

### Get User Permissions

**`GET /users/{userID}/permissions`** — _Requires auth + `rbac.users.view`_

**Response `200`:** `[]Permission`

---

### Assign Role to User

**`POST /users/{userID}/roles`** — _Requires auth + `rbac.users.manage`_

**Request body:**

```json
{
  "role_id": "uuid"
}
```

**Response `200`:**

```json
{ "message": "role assigned successfully" }
```

---

### Remove Role from User

**`DELETE /users/{userID}/roles/{roleID}`** — _Requires auth + `rbac.users.manage`_

**Response `200`:**

```json
{ "message": "role removed successfully" }
```

---

## Error Responses

All error responses use the following JSON shape:

```json
{
  "error": "short_error_code",
  "message": "Human-readable description of what went wrong"
}
```

Some simple errors only include the `error` field.

### Common HTTP status codes

| Status | Meaning                                                                 |
| ------ | ----------------------------------------------------------------------- |
| `400`  | Bad request — invalid input or missing required fields                  |
| `401`  | Unauthorized — missing or invalid token                                 |
| `403`  | Forbidden — valid token but insufficient permissions, or account banned |
| `404`  | Not found                                                               |
| `409`  | Conflict — resource already exists (e.g. duplicate email/handle)        |
| `500`  | Internal server error                                                   |

---

## Submission Types

Submissions have two types which affect statistics:

| Type         | Description                                                                                    |
| ------------ | ---------------------------------------------------------------------------------------------- |
| `test_run`   | User tests their code — **not** counted toward acceptance rate or `is_solved`                  |
| `submission` | Actual attempt — counted toward acceptance rate; marks problem as solved on `Accepted` verdict |
