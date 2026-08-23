package store

// OperationType identifies a database operation compactly.
type OperationType uint8

const (
	OpGet OperationType = iota
	OpSet
	OpDelete
	OpInvalid OperationType = 255
)

// WriteRequest is a mutation submitted to the store's write pipeline.
type WriteRequest struct {
	Operation OperationType
	Key       string
	Value     string
}
