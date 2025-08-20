package events

// Team Event Types
const (
	TeamCreated     = "TEAM_CREATED"
	MemberAdded     = "MEMBER_ADDED"
	MemberRemoved   = "MEMBER_REMOVED"
	ManagerAdded    = "MANAGER_ADDED"
	ManagerRemoved  = "MANAGER_REMOVED"
)

// Asset Event Types
const (
	FolderCreated   = "FOLDER_CREATED"
	FolderUpdated   = "FOLDER_UPDATED"
	FolderDeleted   = "FOLDER_DELETED"
	FolderShared    = "FOLDER_SHARED"
	FolderUnshared  = "FOLDER_UNSHARED"
	
	NoteCreated     = "NOTE_CREATED"
	NoteUpdated     = "NOTE_UPDATED"
	NoteDeleted     = "NOTE_DELETED"
	NoteShared      = "NOTE_SHARED"
	NoteUnshared    = "NOTE_UNSHARED"
)

// Kafka Topics
const (
	TeamActivityTopic = "team.activity"
	AssetChangesTopic = "asset.changes"
)

// Asset Types
const (
	AssetTypeFolder = "folder"
	AssetTypeNote   = "note"
)