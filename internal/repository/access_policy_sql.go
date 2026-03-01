package repository

import "fmt"

func activeProjectMembershipExistsSQL(projectColumn string, actorPlaceholder string) string {
	return fmt.Sprintf(
		`
		EXISTS (
			SELECT 1
			FROM project_members pm
			WHERE
				pm.project_id = %s
				AND pm.user_id = %s
				AND pm.revoked_at IS NULL
		)
		`,
		projectColumn,
		actorPlaceholder,
	)
}

func actorIsAdminSQL(actorPlaceholder string) string {
	return fmt.Sprintf(
		`
		EXISTS (
			SELECT 1
			FROM users actor_user
			WHERE
				actor_user.user_id = %s
				AND actor_user.role = 'admin'
				AND actor_user.is_active = TRUE
		)
		`,
		actorPlaceholder,
	)
}

func buildMembershipReadClause(
	ownerColumn string,
	visibilityColumn string,
	projectColumn string,
	actorPlaceholder string,
	includeOwnerless bool,
) string {
	membershipClause := activeProjectMembershipExistsSQL(projectColumn, actorPlaceholder)
	ownerlessClause := ""
	if includeOwnerless {
		ownerlessClause = fmt.Sprintf(" OR (%s IS NULL AND %s)", ownerColumn, membershipClause)
	}
	return fmt.Sprintf(
		"(%s = %s OR %s OR (%s = 'project' AND %s)%s)",
		ownerColumn,
		actorPlaceholder,
		actorIsAdminSQL(actorPlaceholder),
		visibilityColumn,
		membershipClause,
		ownerlessClause,
	)
}

func buildMembershipWriteClause(projectColumn string, actorPlaceholder string) string {
	return fmt.Sprintf(
		"(%s OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = %s AND pm.user_id = %s AND pm.revoked_at IS NULL AND pm.role IN ('owner', 'editor')))",
		actorIsAdminSQL(actorPlaceholder),
		projectColumn,
		actorPlaceholder,
	)
}
