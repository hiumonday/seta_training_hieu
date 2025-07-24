# SETA Training - Microservices User, Team & Asset Management System

A complete microservices system that provides user management through a **Node.js GraphQL service** and team/asset management through a **Go REST API service**. Both services share JWT authentication and a PostgreSQL database.

## 🏗 System Architecture

### **Dual Service Architecture:**

- **Node.js GraphQL Service (Port 4000)**: User management, authentication, JWT generation
- **Go REST API Service (Port 8080)**: Team management, asset management, JWT validation

### **Shared Resources:**

- **PostgreSQL Database**: Shared schema with Node.js managing Users/Teams/Rosters, Go managing Assets
- **JWT Authentication**: Node.js generates tokens, Go validates them
- **User Roles**: Manager (can create teams) vs Member (team participants only)

## 🚀 Features

### 📊 User Management (Node.js GraphQL)

- ✅ Create users with roles (manager/member)
- ✅ JWT-based authentication (login/logout)
- ✅ User queries and listing
- ✅ Password hashing with bcrypt
- ✅ Token generation with ACCESS_TOKEN_SECRET

### 👥 Team Management (Go REST API)

- ✅ Create teams (managers only)
- ✅ Add/remove members to/from teams
- ✅ Add/remove managers (with hierarchy controls)
- ✅ Role-based access control
- ✅ Team roster management with leader/member distinction

### 📁 Asset Management (Go REST API)

- ✅ Create/manage folders and notes
- ✅ Share folders/notes with read/write access levels
- ✅ Revoke sharing permissions
- ✅ Hierarchical permissions (folder sharing includes notes)
- ✅ Manager oversight of team assets
- ✅ User asset browsing for managers

## 🛠 Tech Stack

### Go REST API Service

- **Framework**: Gin web framework
- **ORM**: GORM with PostgreSQL
- **Authentication**: JWT validation middleware
- **UUID**: Google UUID for primary keys

### Node.js GraphQL Service

- **GraphQL**: Apollo Server
- **ORM**: Sequelize with PostgreSQL
- **Authentication**: JWT generation with jsonwebtoken
- **Password**: bcrypt hashing

### Database

- **PostgreSQL**: Shared database with coordinated schema
- **Node.js Tables**: Users (UUID), Teams (INTEGER), Rosters (junction table)
- **Go Tables**: Folders, Notes, FolderShares, NoteShares (UUID)

## 📋 Prerequisites

- **Go 1.23+**
- **Node.js 18+**
- **PostgreSQL 13+**
- **Git**

## 🔧 Setup Instructions

### 1. Install Dependencies

```bash
# Go dependencies
go mod tidy

# If you have Node.js service:
# npm install (in Node.js service directory)
```

### 2. Run Services

**Start Node.js GraphQL Service first (Port 4000):**

```bash
# In Node.js service directory
npm start
```

**Start Go REST API Service (Port 8080):**

```bash
go run cmd/server/main.go
```

## 🔗 Service Integration

### Authentication Flow

1. **User Registration/Login**: Use Node.js GraphQL service
2. **Get JWT Token**: From GraphQL login mutation
3. **API Calls**: Include `Authorization: Bearer <token>` header for Go REST endpoints

### API Endpoints

#### Node.js GraphQL (Port 4000)

```graphql
# User Management
mutation CreateUser($username: String!, $email: String!, $role: String!) {
  createUser(username: $username, email: $email, role: $role) {
    userId
    username
    email
    role
  }
}

mutation Login($email: String!, $password: String!) {
  login(email: $email, password: $password) {
    accessToken
    refreshToken
    user {
      userId
      username
      role
    }
  }
}

query FetchUsers {
  fetchUsers {
    userId
    username
    email
    role
  }
}
```

#### Go REST API (Port 8080)

**Team Management:**

```bash
# Create team (Manager only)
POST /api/teams
Authorization: Bearer <jwt-token>
{
  "teamName": "Development Team",
  "managers": [{"managerId": "uuid"}],
  "members": [{"memberId": "uuid"}]
}

# Get team details
GET /api/teams/:teamId
Authorization: Bearer <jwt-token>

# Add member to team
POST /api/teams/:teamId/members
Authorization: Bearer <jwt-token>
{
  "memberId": "uuid"
}
```

**Asset Management:**

```bash
# Create folder
POST /api/folders
Authorization: Bearer <jwt-token>
{
  "folderName": "Project Documents"
}

# Create note in folder
POST /api/folders/:folderId/notes
Authorization: Bearer <jwt-token>
{
  "title": "Meeting Notes",
  "content": "Discussion points..."
}

# Share folder
POST /api/folders/:folderId/share
Authorization: Bearer <jwt-token>
{
  "userId": "uuid",
  "accessLevel": "write"
}
```

**Manager Oversight:**

```bash
# View team assets (Manager only)
GET /api/teams/:teamId/assets
Authorization: Bearer <jwt-token>

# View user assets (Manager only)
GET /api/users/:userId/assets
Authorization: Bearer <jwt-token>
```
