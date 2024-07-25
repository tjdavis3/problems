package problems

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

// Basic example
func Example() {
	prob := New(500, "An Error has occurred")
	_ = prob.Set("Title", "Test Error")
	_ = prob.Set("Instance", "/error/test")
	prob.PrettyPrint()
	// Output: {
	//   "detail": "An Error has occurred",
	//   "instance": "/error/test",
	//   "status": 500,
	//   "title": "Test Error",
	//   "type": "about:blank"
	//}
}

// Example with extended attributes.  The type must be
// set to something other than about:blank to add
// extended attributes
func Example_extended() {
	prob := New(500, "An Error has occurred")
	_ = prob.Set("Title", "Test Error")
	_ = prob.Set("Instance", "/error/test")
	_ = prob.Set("Type", "uri:example:extended")
	_ = prob.Set("TraceID", "12345-67890")
	prob.PrettyPrint()
	// Output: {
	//   "detail": "An Error has occurred",
	//   "instance": "/error/test",
	//   "status": 500,
	//   "title": "Test Error",
	//   "traceid": "12345-67890",
	//   "type": "uri:example:extended"
	//}
}

// Example with extended attributes, including an array of
// problem fields.  The type must be
// set to something other than about:blank to add
// extended attributes
func Example_array() {
	prob := New(500, "An Error has occurred")
	_ = prob.Set("Title", "Test Error")
	_ = prob.Set("Instance", "/error/test")
	_ = prob.Set("Type", "uri:example:extended")
	_ = prob.Set("TraceID", "12345-67890")
	issues := make(map[string]interface{})
	issues["field"] = "state"
	issues["message"] = "A valid state must be provided"
	_ = prob.Set("invalid-params", []map[string]interface{}{issues})
	prob.PrettyPrint()
	// Output: {
	//   "detail": "An Error has occurred",
	//   "instance": "/error/test",
	//   "invalid-params": [
	//     {
	//       "field": "state",
	//       "message": "A valid state must be provided"
	//     }
	//   ],
	//   "status": 500,
	//   "title": "Test Error",
	//   "traceid": "12345-67890",
	//   "type": "uri:example:extended"
	// }
}

func TestProblem_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		problem *Problem
		wantErr bool
	}{
		{
			name:    "Test standard problem",
			problem: New(500, "An Error occurred"),
			wantErr: false,
		},
		{
			name:    "Test wrapped error",
			problem: Wrap(fmt.Errorf("This is an error")),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := tt.problem
			if err := prob.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("Problem.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProblem_Error(t *testing.T) {
	type fields struct {
		Type       string
		Title      string
		Status     int
		Detail     string
		Instance   string
		Attributes map[string]interface{}
		err        error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test message",
			fields: fields{
				Detail: "Test Error",
				Status: 500,
			},
			want: "500: Test Error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := &Problem{
				Type:       tt.fields.Type,
				Title:      tt.fields.Title,
				Status:     tt.fields.Status,
				Detail:     tt.fields.Detail,
				Instance:   tt.fields.Instance,
				Attributes: tt.fields.Attributes,
				err:        tt.fields.err,
			}
			if got := prob.Error(); got != tt.want {
				t.Errorf("Problem.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblem_Set(t *testing.T) {
	probFromErr := FromError(fmt.Errorf("Error"))
	type args struct {
		key   string
		value interface{}
	}
	tests := []struct {
		name    string
		problem *Problem
		args    args
		wantErr bool
	}{
		{
			name:    "Test invalid attribute",
			args:    args{key: "TestField", value: "Anything"},
			problem: probFromErr,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := tt.problem

			if err := prob.Set(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Problem.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProblem_Render(t *testing.T) {
	type fields struct {
		Type       string
		Title      string
		Status     int
		Detail     string
		Instance   string
		Attributes map[string]interface{}
		err        error
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := &Problem{
				Type:       tt.fields.Type,
				Title:      tt.fields.Title,
				Status:     tt.fields.Status,
				Detail:     tt.fields.Detail,
				Instance:   tt.fields.Instance,
				Attributes: tt.fields.Attributes,
				err:        tt.fields.err,
			}
			_ = prob.Render(tt.args.w, tt.args.r)
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		status  int
		message string
	}
	tests := []struct {
		name string
		args args
		want *Problem
	}{
		{
			name: "Test 500 error",
			args: args{status: 500, message: "Test Error"},
			want: &Problem{Status: 500, Detail: "Test Error"},
		},
		{
			name: "Test invalid error code",
			args: args{status: 555, message: "Test Error"},
			want: &Problem{Status: 555, Detail: "Test Error", Title: http.StatusText(555)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.status, tt.args.message); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func TestFromError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *Problem
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromError(tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *Problem
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Wrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblem_PrettyPrint(t *testing.T) {
	type fields struct {
		Type       string
		Title      string
		Status     int
		Detail     string
		Instance   string
		Attributes map[string]interface{}
		err        error
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := &Problem{
				Type:       tt.fields.Type,
				Title:      tt.fields.Title,
				Status:     tt.fields.Status,
				Detail:     tt.fields.Detail,
				Instance:   tt.fields.Instance,
				Attributes: tt.fields.Attributes,
				err:        tt.fields.err,
			}
			prob.PrettyPrint()
		})
	}
}
