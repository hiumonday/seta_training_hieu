# Copilot Instructions for SETA Training System

## Architecture Overview

This is a **microservices system** with dual API architecture:

- **Node.js GraphQL Service**: User management (auth, registration, user queries)
- **Go REST API Service**: Team/asset operations (folders, notes, sharing)

Both services share JWT authentication and PostgreSQL database.

Core entities: `User` → `Team` (many-to-many) → `Folder` → `Note` with granular sharing permissions.

## Key Patterns & Conventions

### Handler Structure

All handlers follow this pattern in `internal/handlers/`:

```go
type Handler struct { db *gorm.DB }
func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }
```

Handlers extract user context from Gin middleware:

```go
userID, exists := c.Get("user_id")  // from JWT token
role, exists := c.Get("role")       // "manager" or "member"
```

### Authorization Patterns

- **Role-based**: Managers can create teams, access team assets
- **Ownership-based**: Only owners can delete folders/notes
- **Access-level sharing**: `Read`/`Write` permissions for folders and notes
- **Hierarchical**: Folder sharing grants access to contained notes

### Database & Models

- Uses GORM with PostgreSQL, UUIDs as primary keys
- **Shared database schema**: Users, Teams, Rosters tables managed by Node.js service
- **Go-managed tables**: Folders, Notes, FolderShares, NoteShares
- Models in `internal/models/models.go` with embedded timestamps
- Custom enums: `UserRole` (manager/member), `AccessLevel` (read/write)
- Junction table: `Rosters` with role field for team membership

### Transaction Pattern for Complex Operations

Critical operations use database transactions:

```go
tx := h.db.Begin()
// ... multiple operations
if err != nil {
    tx.Rollback()
    return
}
tx.Commit()
```

## Development Workflows

### Service Integration Setup

```bash
# Both services must use same JWT_SECRET and database
# Node.js GraphQL Service (port 4000): User management, authentication
#   - Endpoint: http://localhost:4000/users (GraphQL)
#   - Handles: createUser, login, logout, fetchUsers
# Go REST API Service (port 8080): Team/Asset management
#   - All endpoints require JWT token from Node.js service
```

### Running the Go REST API

```bash
# Setup environment
cp .env.example .env  # Configure DB credentials + JWT_SECRET
go mod tidy

# Run server
go run cmd/server/main.go  # Starts on :8080
```

### Testing API Endpoints

- Node.js GraphQL: `POST http://localhost:4000/users` for user operations
- Use JWT token from GraphQL login mutation for REST API calls
- REST endpoints require `Authorization: Bearer <jwt-token>` header
- Examples in `API_EXAMPLES.md` for cURL commands

### Key Environment Variables

```env
DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME  # PostgreSQL config
JWT_SECRET  # Token signing key
PORT=8080   # Server port
```

## Critical Business Logic

### Access Control Helpers

In `asset_handler.go`, these methods implement complex permission checking:

- `checkFolderAccess()` - Owner OR shared user
- `checkFolderWriteAccess()` - Owner OR write-level share
- `checkNoteAccess()` - Direct note access OR folder access (hierarchical)

### Team Management Rules

- Only managers can create teams
- Team creators become initial managers
- Only team managers can add/remove members
- Only team creator can add/remove other managers
- Manager-only endpoints: `/api/teams/:teamId/assets`, `/api/users/:userId/assets`

### Sharing System

- Folders and notes have independent sharing (both can be shared separately)
- Share updates existing permissions rather than creating duplicates
- Cascading deletes: Deleting folder removes contained notes and all shares

## Common Gotchas

1. **UUID Parsing**: All route parameters need `uuid.Parse()` validation
2. **Role Checking**: Use string comparison `role != string(models.Manager)`
3. **Transaction Cleanup**: Always handle rollback on errors in complex operations
4. **Access Inheritance**: Notes inherit folder permissions but can have additional direct shares
5. **GORM Preloading**: Use `.Preload()` for relationships in manager oversight endpoints

## File Organization

- `cmd/server/main.go` - Entry point, route definitions
- `internal/handlers/` - Business logic (team_handler.go, asset_handler.go)
- `internal/models/` - Data models and enums
- `internal/middleware/` - JWT authentication
- `internal/auth/` - Token generation/validation
- `internal/database/` - DB connection and auto-migration

# seta-training

SETA golang/nodejs training

# 🏗 Training Exercise: User, Team & Asset Management

## 🎯 Objective

Build a system to manage users and teams:

- Users can have roles: **manager** or **member**.
- Managers can create teams, add/remove members or other managers.
- Users can manage and share digital assets (folders & notes) with access control.

---

## ⚙ System Architecture

- ✅ **GraphQL service**: For user management: create user, login, logout, fetch users, assign roles.
- ✅ **REST API**: For team management & asset management (folders, notes, sharing).

---

## 🧩 Functional Requirements

### 🔹 User Management (GraphQL)

- Create user:
  - `userId` (auto-generated)
  - `username`
  - `email` (unique)
  - `role`: "manager" or "member"
- Authentication:
  - Login, logout (JWT or session-based)
- User listing & query:
  - `fetchUsers` to get list of users
- Role assignment:
  - Manager: can create teams, manage users in teams
  - Member: can only be added to teams, no team management

---

### 🔹 Team Management (REST API)

- Managers can:
  - Create teams
  - Add/remove members
  - Add/remove other managers (only main manager can do this)

Each team:

- `teamId`
- `teamName`
- `managers` (list)
- `members` (list)

---

### 🔹 Asset Management & Sharing (REST API)

- **Folders**: owned by users, contain notes
- **Notes**: belong to folders, have content
- Users can:
  - Share folders or individual notes with other users (read or write access)
  - Revoke access at any time
- When sharing a folder → all notes inside are also shared

**Managers**:

- Can view (read-only) all assets their team members have or can access
- Cannot edit unless explicitly shared with write access

---

## 🔑 Key Rules & Permissions

- Only authenticated users can use APIs.
- Managers can only manage users within their own teams.
- Members cannot create/manage teams.
- Only asset owners can manage sharing.

---

## 🛠 API Endpoints

### 📌 GraphQL: User Management

| Query/Mutation                      | Description             |
| ----------------------------------- | ----------------------- |
| `createUser(username, email, role)` | Create a new user       |
| `login(email, password)`            | Login and receive token |
| `logout()`                          | Logout current user     |
| `fetchUsers()`                      | List all users          |

---

### 📌 REST API: Team Management

| Method | Path                                 | Description        |
| ------ | ------------------------------------ | ------------------ |
| POST   | /teams                               | Create a team      |
| POST   | /teams/{teamId}/members              | Add member to team |
| DELETE | /teams/{teamId}/members/{memberId}   | Remove member      |
| POST   | /teams/{teamId}/managers             | Add manager        |
| DELETE | /teams/{teamId}/managers/{managerId} | Remove manager     |

#### ✅ Create team – request body:

```json
{
  "teamName": "string",
  "managers": [{ "managerId": "string", "managerName": "string" }],
  "members": [{ "memberId": "string", "memberName": "string" }]
}
```

---

### 📌 REST API: Asset Management

#### 🔹 Folder Management

| Method | Path                | Description                    |
| ------ | ------------------- | ------------------------------ |
| POST   | /folders            | Create new folder              |
| GET    | /folders/\:folderId | Get folder details             |
| PUT    | /folders/\:folderId | Update folder (name, metadata) |
| DELETE | /folders/\:folderId | Delete folder and its notes    |

#### 🔹 Note Management

| Method | Path                      | Description               |
| ------ | ------------------------- | ------------------------- |
| POST   | /folders/\:folderId/notes | Create note inside folder |
| GET    | /notes/\:noteId           | View note                 |
| PUT    | /notes/\:noteId           | Update note               |
| DELETE | /notes/\:noteId           | Delete note               |

#### 🔹 Sharing API

| Method | Path                               | Description                         |
| ------ | ---------------------------------- | ----------------------------------- |
| POST   | /folders/\:folderId/share          | Share folder with user (read/write) |
| DELETE | /folders/\:folderId/share/\:userId | Revoke folder sharing               |
| POST   | /notes/\:noteId/share              | Share single note                   |
| DELETE | /notes/\:noteId/share/\:userId     | Revoke note sharing                 |

#### 🔹 Manager-only APIs

| Method | Path                   | Description                                         |
| ------ | ---------------------- | --------------------------------------------------- |
| GET    | /teams/\:teamId/assets | View all assets that team members own or can access |
| GET    | /users/\:userId/assets | View all assets owned by or shared with user        |

---

## 🧩 Database Design Suggestion (PostgreSQL)

- Users: `userId`, `username`, `email`, `role`, `passwordHash`
- Teams: `teamId`, `teamName`
- team_members, team_managers: mapping tables
- Folders: `folderId`, `name`, `ownerId`
- Notes: `noteId`, `title`, `body`, `folderId`, `ownerId`
- folder_shares, note_shares: `userId`, `access` ("read" or "write")

---

## ✅ Development Requirements

- Use JWT for authentication
- Validate role before allowing team creation or manager addition
- Handle errors: duplicate email, invalid role, unauthorized actions
- Write models for User, Team, Folder, Note
- Use Go Framework (Gin + GORM)

---
