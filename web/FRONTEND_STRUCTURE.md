# Frontend Structure

## Overview
Clean, production-ready authentication system with Mantine UI and Tailwind CSS.

## Directory Structure

```
src/
├── api/
│   ├── client.ts          # HTTP client with auth token management
│   └── auth.ts            # Auth API endpoints (register, login, logout, me)
├── components/
│   ├── auth/
│   │   ├── LoginModal.tsx    # Login modal with form validation
│   │   └── RegisterModal.tsx # Register modal with form validation
│   └── layout/
│       └── Navbar.tsx        # Top navigation with search, auth buttons, user menu
├── contexts/
│   └── AuthContext.tsx    # Auth state management and API integration
├── types/
│   └── auth.ts            # TypeScript types matching backend models
├── App.tsx                # Main app component
├── main.tsx               # App entry point with providers
└── index.css              # Global styles with Tailwind + Mantine

```

## Features Implemented

### ✅ Authentication System
- **Register**: Modal with handle, email, password, display name (optional)
- **Login**: Modal with email and password
- **Logout**: Clears tokens and user state
- **Get Current User**: Fetches user data on app load if token exists
- **Token Management**: Automatic storage and retrieval from localStorage

### ✅ Navigation Bar
- **Logo**: Left side (Capstone)
- **Search Bar**: Center (placeholder for future implementation)
- **Auth Buttons**: Right side when not logged in (Login, Register)
- **User Avatar**: Right side when logged in
  - Shows avatar image or first letter of display name/handle
  - Dropdown menu with:
    - User info (display name, handle)
    - Profile link (placeholder)
    - Logout button

### ✅ Type Safety
All backend types are mirrored in TypeScript:
- `User` - matches backend User model
- `AuthResponse` - matches backend auth response
- `RegisterRequest` - matches backend register DTO
- `LoginRequest` - matches backend login DTO
- `UserStatus` - enum for ACTIVE/BANNED

### ✅ API Integration
- Base URL configured via environment variable (`VITE_API_URL`)
- Automatic token injection in requests
- Error handling with user-friendly notifications
- Endpoints:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/auth/me`
  - `POST /api/v1/auth/refresh`

### ✅ Form Validation
- Email format validation
- Password length validation (min 8 characters)
- Required field validation
- Real-time error messages

### ✅ User Experience
- Mantine modals for auth forms
- Toast notifications for success/error feedback
- Loading states during API calls
- Persistent authentication across page refreshes
- Smooth hover effects and transitions

## Environment Variables

Create `.env` file:
```env
VITE_API_URL=http://localhost:3000
```

## Running the App

```bash
# Development
npm run dev

# Build
npm run build

# Preview production build
npm run preview
```

## Backend Integration

The frontend expects the backend API to be running on `http://localhost:3000` (configurable via `.env`).

### Required Backend Endpoints:
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/logout` - Logout user
- `GET /api/v1/auth/me` - Get current user (requires auth)
- `POST /api/v1/auth/refresh` - Refresh access token

### CORS Configuration
Make sure your backend allows requests from `http://localhost:5174` (or your dev server port).

## Tech Stack

- **React 19** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool
- **Mantine 7** - Component library
- **Tailwind CSS** - Utility CSS
- **Tabler Icons** - Icon library

## Next Steps

1. Add routing (React Router)
2. Add protected routes
3. Implement search functionality
4. Add profile page
5. Add token refresh logic
6. Add remember me functionality
7. Add password reset flow
8. Add email verification flow
