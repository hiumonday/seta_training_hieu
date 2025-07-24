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

## 📊 Database Schema

### Node.js Managed Tables

```sql
-- Users table (managed by Node.js)
CREATE TABLE "Users" (
  "userId" UUID PRIMARY KEY,
  "username" VARCHAR(255) NOT NULL,
  "email" VARCHAR(255) UNIQUE NOT NULL,
  "password" VARCHAR(255) NOT NULL,
  "role" VARCHAR(50) CHECK (role IN ('MANAGER', 'MEMBER'))
);

-- Teams table (managed by Node.js)
CREATE TABLE "Teams" (
  "teamId" SERIAL PRIMARY KEY,
  "teamName" VARCHAR(255) NOT NULL
);

-- Rosters junction table (managed by Node.js)
CREATE TABLE "Rosters" (
  "teamId" INTEGER REFERENCES "Teams"("teamId"),
  "userId" UUID REFERENCES "Users"("userId"),
  "isLeader" BOOLEAN DEFAULT FALSE,
  PRIMARY KEY ("teamId", "userId")
);
```

### Go Managed Tables

```sql
-- Folders (managed by Go)
CREATE TABLE folders (
  id UUID PRIMARY KEY,
  folder_name VARCHAR(255) NOT NULL,
  owner_id UUID REFERENCES "Users"("userId"),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Notes (managed by Go)
CREATE TABLE notes (
  id UUID PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  content TEXT,
  folder_id UUID REFERENCES folders(id),
  owner_id UUID REFERENCES "Users"("userId"),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Sharing tables (managed by Go)
CREATE TABLE folder_shares (
  folder_id UUID REFERENCES folders(id),
  user_id UUID REFERENCES "Users"("userId"),
  access_level VARCHAR(10) CHECK (access_level IN ('read', 'write')),
  PRIMARY KEY (folder_id, user_id)
);
```

## 🔐 Security & Authorization

### Role-Based Access Control

- **Managers**: Can create teams, manage team members, view team assets
- **Members**: Can be added to teams, manage own assets, receive shared assets

### Permission Levels

- **Folder/Note Ownership**: Full control (edit, delete, share)
- **Write Access**: Edit content, cannot delete or share
- **Read Access**: View only, cannot modify

### JWT Token Security

- **Token Generation**: Node.js service only
- **Token Validation**: Go service validates using shared secret
- **Token Claims**: userId, role extracted for authorization

## 🧪 Testing

### cURL Examples

**1. Register User (Node.js GraphQL):**

```bash
curl -X POST http://localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { createUser(username: \"john\", email: \"john@example.com\", role: \"MANAGER\") { userId username role } }"
  }'
```

**2. Login (Node.js GraphQL):**

```bash
curl -X POST http://localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { login(email: \"john@example.com\", password: \"password\") { accessToken user { userId role } } }"
  }'
```

**3. Create Team (Go REST):**

```bash
curl -X POST http://localhost:8080/api/teams \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "teamName": "Development Team",
    "managers": [],
    "members": []
  }'
```

**4. Create Folder (Go REST):**

```bash
curl -X POST http://localhost:8080/api/folders \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "folderName": "Project Documents"
  }'
```

## 🚨 Troubleshooting

### Common Issues

**1. JWT Token Mismatch:**

- Ensure both services use same `JWT_SECRET` in environment
- Verify token format from Node.js login matches Go validation

**2. Database Connection:**

- Check PostgreSQL is running and accessible
- Verify database credentials in `.env` file
- Ensure database exists and tables are created

**3. CORS Issues:**

- Add CORS middleware if accessing from browser
- Configure allowed origins for cross-service communication

**4. Permission Denied:**

- Verify user role (only managers can create teams)
- Check JWT token is included in Authorization header
- Confirm user is team manager for team operations

### Debug Commands

**Check Go service compilation:**

```bash
go build ./cmd/server
```

**View database tables:**

```sql
-- Connect to PostgreSQL
psql -h localhost -U your_username -d seta_training

-- List tables
\dt

-- Check Users table
SELECT * FROM "Users";

-- Check team rosters
SELECT * FROM "Rosters";
```

**Test JWT token validation:**

```bash
# Decode JWT token (online: jwt.io)
echo "your-jwt-token" | base64 -d
```

## 📚 Development Guidelines

### Code Organization

- **cmd/server/main.go**: Entry point, route registration
- **internal/handlers/**: Business logic (team_handler.go, asset_handler.go)
- **internal/models/**: Database models and enums
- **internal/middleware/**: JWT authentication middleware
- **internal/auth/**: Token validation utilities
- **internal/database/**: Database connection and auto-migration

### Adding New Features

1. **Models**: Define in `internal/models/models.go`
2. **Handlers**: Create methods in appropriate handler
3. **Routes**: Register in `cmd/server/main.go`
4. **Middleware**: Add authentication where needed
5. **Testing**: Test with cURL or API client

### Database Changes

- **Go tables**: Modify models and restart service (auto-migration)
- **Node.js tables**: Coordinate with Node.js service team
- **Schema conflicts**: Go service adapts to Node.js schema

## 🎯 Next Steps

### Completed ✅

- [x] Microservices architecture with Node.js + Go
- [x] Shared PostgreSQL database with coordinated schema
- [x] JWT authentication integration
- [x] Complete team management with Roster junction table
- [x] Asset management with sharing permissions
- [x] Manager oversight capabilities
- [x] Role-based access control

### Potential Enhancements 🚀

- [ ] React frontend for complete user interface
- [ ] Real-time notifications for sharing events
- [ ] File upload support for assets
- [ ] Advanced search and filtering
- [ ] Audit logging for security
- [ ] API rate limiting
- [ ] Docker containerization
- [ ] Kubernetes deployment

## 📄 License

This project is part of SETA training curriculum for microservices development.

---

**🎓 SETA Training System - Complete Microservices Integration Achieved!**

_Node.js GraphQL + Go REST API + PostgreSQL + JWT Authentication_
