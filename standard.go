package problems

import (
	"fmt"
	"net/http"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// Problem types
const (
	TypeNoAccessToken       = "urn:problem-type:noAccessToken"
	TypeInvalidToken        = "urn:problem-type:invalidAccessToken"
	TypeTokenExpired        = "urn:problem-type:expiredAccessToken"
	TypeMissingScope        = "urn:problem-type:missingScope"
	TypeMissingPermission   = "urn:problem-type:missingPermission"
	TypeNotFound            = "urn:problem-type:resourceNotFound"
	TypeBadRequest          = "urn:problem-type:badRequest"
	TypeSchemaViolation     = "urn:problem-type:input-validation:schemaViolation"
	TypeUnknownParameter    = "urn:problem-type:input-validation:unknownParameter"
	TypeInternalServerError = "urn:problem-type:internalServerError"
	TypeConflict            = "urn:problem-type:conflict"
)

var validate *validator.Validate
var uni *ut.UniversalTranslator
var trans ut.Translator

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
	en := en.New()
	uni = ut.New(en, en)
	trans, _ = uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(validate, trans)

}

func GetNoAccessResponse() *Problem {
	prob := New(401, "No Bearer access token found in Authorization HTTP header")
	prob.Type = TypeNoAccessToken
	prob.Set("Title", "No Access Token")
	return prob
}

func GetInvalidTokenResponse() *Problem {
	prob := New(401, "The Bearer access token found in the Authorization HTTP header is invalid")
	prob.Set("Title", "Invalid Access Token")
	prob.Set("Type", TypeInvalidToken)
	return prob
}

func GetExpiredTokenResponse() *Problem {
	prob := New(401, "The Bearer access token found in the Authorization HTTP header has expired")
	prob.Set("Type", TypeTokenExpired)
	prob.Set("Title", "Expired Access Token")
	return prob
}

func GetMissingScopeResponse(scopes []string) *Problem {
	prob := New(403, "Forbidden to consult the resource")
	prob.Set("Type", TypeMissingScope)
	prob.Set("Title", "Missing Scope")

	prob.Set("requiredScopes", scopes)
	return prob
}

func GetMissingPermission() *Problem {
	prob := New(403, "Not permitted to update the details of this resource")
	prob.Set("Type", TypeMissingPermission)
	prob.Set("Title", "Missing Permission")
	return prob
}

func GetInternalErrorResponse(detail string) *Problem {
	prob := New(http.StatusInternalServerError, detail)
	prob.Set("Type", TypeInternalServerError)
	prob.Set("Title", http.StatusText(http.StatusInternalServerError))
	return prob
}

func GetErrorResponseFromError(err error) *Problem {
	prob := FromError(err)
	prob.Set("Type", TypeInternalServerError)
	return prob
}

// MissingResourceParam is passed to GetMissingResource to set the problem values
type MissingResourceParam struct {
	// ResourceType is the type of resource that is missing, eg. User
	ResourceType string
	// ResourceValue is the value that was requested
	ResourceValue interface{}
	// Location indicates where the resource was called from (path/body)
	Location string
	// Location is the API path that was called (eg. /users/123)
	URL string
}

// GetMissingResource creates a Problem that defines the resource that was not found
func GetMissingResource(resource MissingResourceParam) *Problem {
	prob := New(404, "Missing Resource")
	prob.Set("Type", TypeNotFound)
	prob.Set("Title", "Resource not found")
	prob.Set("Detail", fmt.Sprintf("No resource %s:%s found", resource.ResourceType, resource.ResourceValue))

	issue := Problem{}
	issue.Set("Type", TypeNotFound)
	if resource.Location != "" {
		issue.Set("in", resource.Location)
	}
	issue.Set("name", resource.ResourceType)
	issue.Set("detail", fmt.Sprintf("the %s %v is not assigned", resource.ResourceType, resource.ResourceValue))
	issue.Set("value", resource.ResourceValue)
	prob.Set("issues", []interface{}{issue})
	return prob
}

type ValidationParam struct {
	Location  string      // path or body
	Name      string      // the field name in error
	Value     interface{} // the value that was provided
	Issue     string      // the problem with the field
	IsUnknown bool        // if the parameter is not defined by the API
}

func GetInputValidationResponse(validations ...ValidationParam) *Problem {
	prob := New(400, "The input message is incorrect; see issues for more information")
	prob.Set("Type", TypeBadRequest)
	prob.Set("Title", "Bad Request")

	if len(validations) == 1 {
		prob.Set("Detail", validations[0].Issue)
	}

	issues := make([]Problem, 0)

	for _, validation := range validations {
		issue := Problem{}
		if validation.IsUnknown {
			issue.Set("Type", TypeUnknownParameter)
		} else {
			issue.Set("Type", TypeSchemaViolation)
		}
		issue.Set("in", validation.Location)
		issue.Set("name", validation.Name)
		issue.Set("value", validation.Value)
		issue.Set("Detail", validation.Issue)
		issues = append(issues, issue)
	}

	prob.Set("issues", issues)
	return prob
}

func GetValidatorResponse(err validator.ValidationErrors) *Problem {
	var errors []ValidationParam
	for _, err := range err {
		param := ValidationParam{
			Location: "body",
			Name:     err.Namespace(),
			Value:    err.Value(),
			Issue:    err.Translate(trans),
		}
		errors = append(errors, param)
	}

	if len(errors) == 0 {
		return nil
	}
	return GetInputValidationResponse(errors...)
}
