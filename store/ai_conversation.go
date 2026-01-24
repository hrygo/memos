package store

type AIConversation struct {
	ID        int32
	UID       string
	CreatorID int32
	Title     string
	ParrotID  string
	Pinned    bool
	CreatedTs int64
	UpdatedTs int64
	RowStatus RowStatus
}

type FindAIConversation struct {
	ID        *int32
	UID       *string
	CreatorID *int32
	Pinned    *bool
	RowStatus *RowStatus
}

type UpdateAIConversation struct {
	ID        int32
	Title     *string
	ParrotID  *string
	Pinned    *bool
	RowStatus *RowStatus
	UpdatedTs *int64
}

type DeleteAIConversation struct {
	ID int32
}

type AIMessageRole string

const (
	AIMessageRoleUser      AIMessageRole = "USER"
	AIMessageRoleAssistant AIMessageRole = "ASSISTANT"
	AIMessageRoleSystem    AIMessageRole = "SYSTEM"
)

type AIMessageType string

const (
	AIMessageTypeMessage   AIMessageType = "MESSAGE"
	AIMessageTypeSeparator AIMessageType = "SEPARATOR"
)

type AIMessage struct {
	ID             int32
	UID            string
	ConversationID int32
	Type           AIMessageType
	Role           AIMessageRole
	Content        string
	Metadata       string // JSON string
	CreatedTs      int64
}

type FindAIMessage struct {
	ID             *int32
	UID            *string
	ConversationID *int32
}

type DeleteAIMessage struct {
	ID             *int32
	ConversationID *int32
}
