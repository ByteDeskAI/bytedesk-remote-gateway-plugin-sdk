package pluginsdk

import "github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"

// Projects contribution contracts are owned by the common SDK and re-exported
// here so Gateway plugin authors import one module.
type (
	ProjectViewContribution                  = plugin.ProjectViewContribution
	DirectoryContextActionContribution       = plugin.DirectoryContextActionContribution
	ProjectDirectoryContext                  = plugin.ProjectDirectoryContext
	DirectoryContextActionEligibilityRequest = plugin.DirectoryContextActionEligibilityRequest
	DirectoryContextActionEligibilityResult  = plugin.DirectoryContextActionEligibilityResult
	DirectoryContextActionWizardContext      = plugin.DirectoryContextActionWizardContext
)

const DirectoryContextActionEligibilityCommand = plugin.DirectoryContextActionEligibilityCommand
