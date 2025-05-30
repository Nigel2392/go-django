package openauth2_models_mysql

import (
	"context"
	"fmt"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	"github.com/Nigel2392/go-django/src/models"
)

const retrieveUsersDynamicOrd = `-- name: RetrieveUsers :many
SELECT id, unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, created_at, updated_at, is_administrator, is_active FROM oauth2_users
ORDER BY %s
LIMIT ?
OFFSET ?
`

func (q *Queries) RetrieveUsers(ctx context.Context, limit int32, offset int32, orderings ...string) ([]*openauth2models.User, error) {

	var orderer = models.Orderer{
		Fields:   orderings,
		Validate: openauth2models.IsValidField,
		Default:  "-" + openauth2models.FieldUpdatedAt,
	}

	ordering, err := orderer.Build()
	if err != nil {
		return nil, err
	}

	var query = fmt.Sprintf(retrieveUsersDynamicOrd, ordering)
	rows, err := q.query(ctx, nil, query,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*openauth2models.User
	for rows.Next() {
		var i openauth2models.User
		if err := rows.Scan(
			&i.ID,
			&i.UniqueIdentifier,
			&i.ProviderName,
			&i.Data,
			&i.AccessToken,
			&i.RefreshToken,
			&i.TokenType,
			&i.ExpiresAt,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
