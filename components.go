package pluginsdk

import "github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"

// Component wire contracts belong to the common SDK.
type (
	ComponentIdentity             = plugin.ComponentIdentity
	ComponentAssignment           = plugin.ComponentAssignment
	ComponentExtension            = plugin.ComponentExtension
	ComponentWorkspaceSnapshot    = plugin.ComponentWorkspaceSnapshot
	ComponentSessionListSnapshot  = plugin.ComponentSessionListSnapshot
	ComponentSessionTabSnapshot   = plugin.ComponentSessionTabSnapshot
	ComponentTerminalSnapshot     = plugin.ComponentTerminalSnapshot
	ComponentPlacement            = plugin.ComponentPlacement
	ComponentStageSnapshot        = plugin.ComponentStageSnapshot
	ComponentTasksSnapshot        = plugin.ComponentTasksSnapshot
	ComponentFileTreeSnapshot     = plugin.ComponentFileTreeSnapshot
	ComponentProjectToolsSnapshot = plugin.ComponentProjectToolsSnapshot
	ComponentAvailableRequest     = plugin.ComponentAvailableRequest
	ComponentAvailableResult      = plugin.ComponentAvailableResult
	ComponentAssignRequest        = plugin.ComponentAssignRequest
	ComponentAssignResult         = plugin.ComponentAssignResult
	ComponentContributeRequest    = plugin.ComponentContributeRequest
	ComponentContributeResult     = plugin.ComponentContributeResult
	ComponentSnapshotRequest      = plugin.ComponentSnapshotRequest
	ComponentSnapshotResult       = plugin.ComponentSnapshotResult
	ComponentInvokeRequest        = plugin.ComponentInvokeRequest
	ComponentChanged              = plugin.ComponentChanged
)

const (
	ComponentAvailableCommand  = plugin.ComponentAvailableCommand
	ComponentAssignCommand     = plugin.ComponentAssignCommand
	ComponentContributeCommand = plugin.ComponentContributeCommand
	ComponentSnapshotCommand   = plugin.ComponentSnapshotCommand
	ComponentInvokeCommand     = plugin.ComponentInvokeCommand
	ComponentChangedEvent      = plugin.ComponentChangedEvent
)

var (
	ValidateComponentIdentity  = plugin.ValidateComponentIdentity
	ValidateComponentExtension = plugin.ValidateComponentExtension
	ComponentMethods           = plugin.ComponentMethods
)
