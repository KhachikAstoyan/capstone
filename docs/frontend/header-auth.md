# Header & Auth

## Overview

The header is a sticky top bar with three sections: logo (left), search input (left), and user area (right). Authentication state is managed by `AuthProvider` and surfaced via the `useAuth()` hook. All API calls go through the typed fetch wrapper in `src/lib/api.ts`.

---

## Environment Setup

Create a `.env.local` file in `capstone-code/web/` (copy from `.env.example`):

```
VITE_API_URL=http://localhost:8080/api/v1
```

The API client reads `import.meta.env.VITE_API_URL` at runtime. If unset it falls back to `/api/v1` (same-origin, useful when the Go server serves the frontend in production).

---

## Components

### `Header` (`src/components/Header.tsx`)

Sticky top bar. Reads auth state from `useAuth()` and renders:

| Section | Content |
|---------|---------|
| Left | `<Link to="/">` with `Code2` icon + "Capstone" text |
| Left | `<Input>` search bar (UI shell — wire to `GET /problems/?search=` when ready) |
| Right (loading) | Pulsing skeleton circle |
| Right (logged in) | `Avatar` → `DropdownMenu` with handle label, Profile link, Log out |
| Right (logged out) | "Log in" (ghost) and "Sign up" (primary) buttons |

Both auth buttons open `AuthModal` with the appropriate default tab.

### `AuthModal` (`src/components/AuthModal.tsx`)

A shadcn `Dialog` wrapping a two-tab `Tabs` panel.

**Props:**

| Prop | Type | Description |
|------|------|-------------|
| `open` | `boolean` | Controls dialog visibility |
| `onOpenChange` | `(open: boolean) => void` | Called on close/open |
| `defaultTab` | `'login' \| 'signup'` | Which tab to show initially |

- Resets form and errors whenever `open` changes to `false`.
- Syncs to `defaultTab` whenever the modal is opened.
- Calls `useAuth().login()` on submit and closes on success; for sign-up, shows a verification notice after successful registration instead of logging in.
- Displays inline `ApiError` messages from the server (e.g. "email already taken", "invalid credentials").

---

## Auth Layer

### `AuthProvider` (`src/lib/auth.tsx`)

Wrap the app root with this provider (already done in `__root.tsx`).

On mount it calls `POST /auth/refresh` using the HttpOnly cookie set by the server. If the cookie is valid the access token and user are stored in memory — no credentials are ever written to `localStorage`.

### `useAuth()` hook

```tsx
const { user, accessToken, loading, login, register, logout } = useAuth()
```

| Field / Method | Type | Description |
|----------------|------|-------------|
| `user` | `User \| null` | Authenticated user, or `null` |
| `accessToken` | `string \| null` | In-memory JWT access token |
| `loading` | `boolean` | `true` while the initial refresh is in flight |
| `login(email, password)` | `Promise<User>` | Calls `POST /auth/login`; throws `ApiError` on failure |
| `register(handle, email, password, displayName?)` | `Promise<void>` | Calls `POST /auth/register` (no session; UI should prompt to verify email); throws `ApiError` on failure |
| `logout()` | `Promise<void>` | Calls `POST /auth/logout`, clears state |

### `User` type

```ts
interface User {
  id: string
  handle: string
  email: string
  email_verified: boolean
  display_name: string
  avatar_url: string
  status: 'ACTIVE' | 'BANNED'
  created_at: string
  updated_at: string
}
```

---

## API Client (`src/lib/api.ts`)

A thin typed `fetch` wrapper. Use it in any feature module that needs to call the backend.

### Helpers

```ts
apiGet<T>(path, options?)           // GET
apiPost<T>(path, body?, options?)   // POST
apiPatch<T>(path, body?, options?)  // PATCH
apiDelete<T>(path, options?)        // DELETE
```

### Example

```ts
import { apiGet, apiPost } from '#/lib/api'

// Fetch problems list
const result = await apiGet<{ problems: Problem[]; total: number }>('/problems/')

// Submit a solution
const submission = await apiPost<Submission>('/submissions/', { problem_id, code, language })
```

### Token Handling

- The client reads the in-memory access token via a callback registered by `AuthProvider`.
- On a `401` response it calls `POST /auth/refresh` once (browser sends the HttpOnly cookie automatically), stores the new access token, and retries the original request.
- If refresh also fails, `onUnauthenticated` is called (which clears auth state) and an `ApiError(401, …)` is thrown.

### `ApiError`

```ts
class ApiError extends Error {
  status: number  // HTTP status code
  message: string // Error message from the server response body
}
```

Always catch `ApiError` separately from generic errors when you want to show server-provided messages to the user.

---

## Token Refresh Strategy

```
page load
  └─ POST /auth/refresh  (cookie sent automatically)
       ├─ 200 → store access_token in memory, user in state
       └─ 4xx → stay logged out

any API call
  └─ 401
       ├─ POST /auth/refresh
       │    ├─ 200 → retry original request with new token
       │    └─ 4xx → clear auth state, throw ApiError(401)
       └─ (other errors propagate as-is)
```

Access tokens are short-lived (1 hour per the API). The HttpOnly refresh cookie lasts 7 days. The user stays logged in across page reloads for up to 7 days with no action required.

---

## Extending the Header

- **Add nav links**: insert `<Link>` elements in the header JSX between the search bar and `ml-auto` user area div.
- **Wire search**: read the `Input` value and navigate to a `/problems` route with `?search=` param.
- **Profile page**: replace the placeholder `<Link to="/">` in the dropdown with the real profile route once it exists.
