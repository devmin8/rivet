package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
)

func ToProjectEnvVarResponse(item database.ProjectEnvVar) dtos.ProjectEnvVarResponse {
	var value *string
	hasValue := item.Value != "" || item.EncryptedValue != ""

	if item.Kind == database.ProjectEnvKindPlain {
		value = &item.Value
		hasValue = true
	}

	return dtos.ProjectEnvVarResponse{
		Key:       item.Key,
		Kind:      dtos.ProjectEnvKind(item.Kind),
		Value:     value,
		HasValue:  hasValue,
		UpdatedAt: item.UpdatedAt,
	}
}

func ToListProjectEnvResponse(items []database.ProjectEnvVar) dtos.ListProjectEnvResponse {
	res := dtos.ListProjectEnvResponse{
		Items: make([]dtos.ProjectEnvVarResponse, 0, len(items)),
	}

	for _, item := range items {
		res.Items = append(res.Items, ToProjectEnvVarResponse(item))
	}

	return res
}
