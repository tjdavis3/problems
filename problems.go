// The problems package represents RFC7807 problem details.
package problems

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

const (
	// ProblemMediaType is the default media type for a Problem response
	ProblemMediaType = "application/problem+json"
)

type renderType string

var jsonType renderType = "json"
var xmlType renderType = "xml"

// Problem is an RFC7807 representation of an error
type Problem struct {
	// Type is a URI reference [RFC3986] that identifies the
	//   problem type.  This specification encourages that, when
	//   dereferenced, it provide human-readable documentation for the
	//   problem type (e.g., using HTML [W3C.REC-html5-20141028]).  When
	//   this member is not present, its value is assumed to be
	//   "about:blank".
	Type string `json:"type,omitempty"`
	// Title is a short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string
	// Status is the HTTP status code ([RFC7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status int
	// Detail is a human-readable explanation specific to this
	// occurrence of the problem.
	Detail string
	// Instance is a URI reference that identifies the specific
	// occurrence of the problem.  It may or may not yield further
	// information if dereferenced.
	Instance string `json:"instance,omitempty"`
	// Attributes are extra fields/data that can be added to the problem.
	// They should be set with the `Set` method.  The `Type` MUST be set
	// and cannot be `about:blank`
	Attributes map[string]interface{} `json:"vars,omitempty" xml:"vars,omitempty"`
	err        error
}

// Error returns a string representation of the problem to meet the Error interface definition
func (prob *Problem) Error() string {
	return fmt.Sprintf("%d: %s", prob.Status, prob.Detail)
}

// Unwrap returns the underlying error or nil
func (prob *Problem) Unwrap() error {
	return prob.err
}

// Set sets the extended attribute identified by key to value
// Setting anything other than the basic attributes requires a type other than `about:blank`
func (prob *Problem) Set(key string, value interface{}) error {
	switch strings.Title(key) {
	case "Title":
		prob.Title = fmt.Sprint(value)
	case "Status":
		return New(500, "Cannot set status with Set")
	case "Type":
		prob.Type = fmt.Sprint(value)
	case "Detail":
		prob.Detail = fmt.Sprint(value)
	case "Instance":
		prob.Instance = fmt.Sprint(value)
	default:
		if prob.Type == "" || prob.Type == "about:blank" {
			err := FromError(prob)
			err.Detail = "Cannot set extended attributes unless Type is set"
			err.PrettyPrint()
			return err
		}
		if prob.Attributes == nil {
			prob.Attributes = make(map[string]interface{})
		}
		prob.Attributes[key] = value
	}
	return nil
}

// Render will output the error as an HTTP response
func (prob *Problem) Render(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", ProblemMediaType)
	if prob.Status != 0 {
		w.WriteHeader(prob.Status)
	}
	return json.NewEncoder(w).Encode(prob)
}

func (prob *Problem) MarshalJSON() ([]byte, error) {
	return prob.Marshal(jsonType)
}

func (prob *Problem) MarshalXML() ([]byte, error) {
	return prob.Marshal(xmlType)
}

func (prob *Problem) Marshal(renderAs renderType) ([]byte, error) {
	out := make(map[string]interface{})
	if prob.Type == "" {
		prob.Type = "about:blank"
	}
	prob.Title = prob.GetTitle()
	subjectValue := reflect.Indirect(reflect.ValueOf(prob))
	subjectType := subjectValue.Type()
	for i := 0; i < subjectType.NumField(); i++ {
		field := subjectType.Field(i)
		name := subjectType.Field(i).Name
		if name != "Attributes" && name != "err" {
			var key string
			var ok bool
			if key, ok = field.Tag.Lookup(string(renderAs)); !ok {
				key = name
			}
			if strings.HasSuffix(key, "omitempty") {
				if len(fmt.Sprintf("%v", subjectValue.FieldByName(name).Interface())) == 0 {
					continue
				}
			}
			key = strings.Split(key, ",")[0]
			key = strings.ToLower(key)
			out[key] = subjectValue.FieldByName(name).Interface()
		}
	}
	if prob.err != nil && prob.Type != "about:blank" {
		if problem, ok := prob.err.(*Problem); ok {
			j, err := problem.MarshalJSON()
			if err == nil {
				out["error"] = j
			}
		}
		out["error"] = prob.err.Error()
	}
	for k, v := range prob.Attributes {
		out[strings.ToLower(k)] = v
	}
	switch renderAs {
	case jsonType:
		return json.Marshal(out)
	case xmlType:
		return xml.Marshal(out)
	default:
		return nil, New(500, "Invalid Marshal type specified")
	}
}

func (prob *Problem) UnmarshalJSON(data []byte) error {
	return prob.Unmarshal(jsonType, data)
}
func (prob *Problem) UnmarshalXML(data []byte) error {
	return prob.Unmarshal(xmlType, data)
}
func (prob *Problem) Unmarshal(renderAs renderType, data []byte) error {
	target := make(map[string]interface{})
	switch renderAs {
	case jsonType:
		if err := json.Unmarshal(data, &target); err != nil {
			return FromError(err)
		}
	case xmlType:
		if err := xml.Unmarshal(data, &target); err != nil {
			return FromError(err)
		}
	default:
		return New(500, fmt.Sprintf("%s is an invalid type", renderAs))
	}
	prob.Attributes = make(map[string]interface{})
	for k, v := range target {
		switch strings.Title(k) {
		case "Type":
			prob.Type = fmt.Sprint(v)
		case "Title":
			prob.Title = fmt.Sprint(v)
		case "Status":
			stat, err := strconv.Atoi(fmt.Sprint(v))
			if err != nil {
				return FromError(err)
			}
			prob.Status = stat
		case "Detail":
			prob.Detail = fmt.Sprint(v)
		case "Instance":
			prob.Instance = fmt.Sprint(v)
		default:
			prob.Attributes[k] = v
		}
	}
	return nil
}

// New initializes a problem
func New(status int, message string) *Problem {
	err := Problem{}
	err.Status = status
	err.Detail = message
	return &err
}

// Title returns the title of the problem or a default
func (p *Problem) GetTitle() string {
	if len(p.Title) == 0 {
		return http.StatusText(p.Status)
	}
	return p.Title
}

// FromErrorWithStatus creates a new problem from the
// provided error but sets the status to the one provided
// rather than the default of 500.
func FromErrorWithStatus(status int, err error) *Problem {
	prob := New(status, err.Error())
	prob.err = fmt.Errorf("Problem: %w", err)
	return prob
}

// FromError creates a new problem from the provided error
func FromError(err error) *Problem {
	return FromErrorWithStatus(http.StatusInternalServerError, err)
}

// Wrap creates a Problem that wraps a standard error
func Wrap(err error) *Problem {
	if problem, ok := err.(*Problem); ok {
		return problem
	}
	prob := FromError(err)
	return prob
}

func (prob *Problem) PrettyPrint() {
	pp, err := json.MarshalIndent(prob, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(pp))

}

// StatusCode returns the status of the problem
func (prob *Problem) StatusCode() int {
	return prob.Status
}
