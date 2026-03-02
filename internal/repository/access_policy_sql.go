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

type membershipReadClauseInput struct {
	ownerColumn      string
	visibilityColumn string
	projectColumn    string
	actorPlaceholder string
	includeOwnerless bool
}

func buildMembershipReadClause(input membershipReadClauseInput) string {
	membershipClause := activeProjectMembershipExistsSQL(input.projectColumn, input.actorPlaceholder)
	ownerlessClause := ""
	if input.includeOwnerless {
		ownerlessClause = fmt.Sprintf(" OR (%s IS NULL AND %s)", input.ownerColumn, membershipClause)
	}
	return fmt.Sprintf(
		"(%s = %s OR %s OR (%s = 'project' AND %s)%s)",
		input.ownerColumn,
		input.actorPlaceholder,
		actorIsAdminSQL(input.actorPlaceholder),
		input.visibilityColumn,
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
