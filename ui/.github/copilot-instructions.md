# Copilot Instructions for SETA Training System UI

<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

## Project Overview

This is the React frontend for the SETA Training System with dual service integration:

- **Node.js GraphQL Service** (port 4000): User authentication and management
- **Go REST API Service** (port 8080): Team and asset management

## Architecture & Integration

### Service Integration Pattern

- Use Apollo Client for GraphQL authentication calls to `http://localhost:4000/users`
- Use Axios for REST API calls to `http://localhost:8080/api/*` with JWT token authentication
- JWT tokens from GraphQL login are used to authenticate REST API requests

### Key Features

1. **Authentication**: Login/Register via GraphQL
2. **Team Management**: Create/manage teams (managers only)
3. **Asset Management**: Folders, notes with hierarchical sharing
4. **Role-based Access**: Manager vs Member permissions
5. **Sharing System**: Read/Write access levels for assets

## Technical Stack

- **React 18** with Vite
- **Apollo Client** for GraphQL
- **Axios** for REST API
- **React Router** for navigation
- **Tailwind CSS** for styling
- **Heroicons** for icons
- **React Hot Toast** for notifications

## Code Patterns

### GraphQL Usage

```javascript
// Authentication mutations
const LOGIN_MUTATION = gql`
  mutation Login($input: LoginInput!) {
    login(input: $input) {
      token
      user {
        id
        username
        role
      }
    }
  }
`;
```

### REST API Usage

```javascript
// Authenticated requests with JWT
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

### State Management

- Use React Context for user authentication state
- Local state for component-specific data
- Apollo Cache for GraphQL data

## File Structure

- `src/components/` - Reusable UI components
- `src/pages/` - Route-based page components
- `src/hooks/` - Custom React hooks
- `src/services/` - API clients (GraphQL + REST)
- `src/context/` - React Context providers
- `src/utils/` - Utility functions

## Development Guidelines

1. **Component Design**: Use functional components with hooks
2. **Styling**: Use Tailwind CSS classes consistently
3. **Error Handling**: Use try/catch with toast notifications
4. **Loading States**: Show loading indicators for async operations
5. **Authentication**: Protect routes based on user role
6. **Responsive Design**: Mobile-first approach with Tailwind

## API Endpoints Reference

### GraphQL (Authentication)

- `createUser` - Register new user
- `login` - Authenticate user
- `fetchUsers` - Get users list

### REST API (Business Logic)

- Teams: `/api/teams/*`
- Assets: `/api/folders/*`, `/api/notes/*`
- Sharing: `/api/folders/:id/share`, `/api/notes/:id/share`
- Manager: `/api/teams/:id/assets`, `/api/users/:id/assets`
