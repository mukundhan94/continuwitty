package projects

import "errors"

var (
	// ErrProjectIDMustNotBeBlank indicates project_id was empty after normalization.
	ErrProjectIDMustNotBeBlank = errors.New("project_id must not be blank")
	// ErrProjectNotFound indicates a project is not visible to the caller.
	ErrProjectNotFound = errors.New("Project not found")
	// ErrUserNotFound indicates no user row matched a default-project update.
	ErrUserNotFound = errors.New("User not found")
	// ErrProjectIDRequiredWhenNoDefaultProject indicates a write call needs explicit project context.
	ErrProjectIDRequiredWhenNoDefaultProject = errors.New("project_id is required when no default project is configured")
	// ErrDefaultProjectNotAccessible indicates a default project cannot be used by the caller.
	ErrDefaultProjectNotAccessible = errors.New("default project is not accessible; set a valid default project first")
	// ErrProjectWriteForbidden indicates actor lacks write access for the selected project.
	ErrProjectWriteForbidden = errors.New("project is not writable by actor")
	// ErrProjectMemberManagementForbidden indicates only owner/admin can manage members.
	ErrProjectMemberManagementForbidden = errors.New("project member management requires owner or admin role")
	// ErrProjectAuditForbidden indicates only owner/admin can view project audit events.
	ErrProjectAuditForbidden = errors.New("project audit visibility requires owner or admin role")
	// ErrProjectMemberNotFound indicates project member row does not exist.
	ErrProjectMemberNotFound = errors.New("Project member not found")
	// ErrProjectOwnerMembershipImmutable indicates owner row cannot be modified via member APIs.
	ErrProjectOwnerMembershipImmutable = errors.New("owner membership cannot be modified via member APIs")
	// ErrProjectOwnerRoleNotAssignable indicates owner role cannot be assigned via member APIs.
	ErrProjectOwnerRoleNotAssignable = errors.New("owner role cannot be assigned via member APIs")
	// ErrEngramNotFound indicates target engram was not found.
	ErrEngramNotFound = errors.New("Engram not found")
	// ErrEngramShareForbidden indicates actor cannot share/unshare this engram.
	ErrEngramShareForbidden = errors.New("actor cannot share or unshare this engram")
)
