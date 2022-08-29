package sdk

// Currently, sdk doesn't have generated constants for cluster status and agent status, declaring our own.
const (
	ClusterStatusReady    = "ready"
	ClusterStatusDeleting = "deleting"
	ClusterStatusDeleted  = "deleted"
	ClusterStatusArchived = "archived"
	ClusterStatusFailed   = "failed"

	ClusterAgentStatusDisconnected  = "disconnected"
	ClusterAgentStatusDisconnecting = "disconnecting"
)
