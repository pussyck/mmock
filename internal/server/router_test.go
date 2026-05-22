package server

import (
	"errors"
	"github.com/jmartin82/mmock/v3/pkg/mock"
	"testing"
)

type DummyMatcher struct {
	OK bool
}

func (dm DummyMatcher) Match(req *mock.Request, mock *mock.Definition, scenarioAware bool) (bool, error) {
	if dm.OK {
		return true, nil
	}
	return false, errors.New("Random Error")
}

type DummyMapper struct {
	mocks []mock.Definition
}

func (mm DummyMapper) Set(URI string, mock mock.Definition) error {
	return nil
}
func (mm DummyMapper) Delete(URI string) error {
	return nil
}
func (mm DummyMapper) Get(URI string) (mock.Definition, bool) {
	return mock.Definition{}, false
}
func (mm DummyMapper) List() []mock.Definition {
	return mm.mocks
}

func TestValidRoute(t *testing.T) {

	mocks := []mock.Definition{
		{
			Response: mock.Response{
				StatusCode: 200,
			},
		},
	}

	dummyMapper := DummyMapper{mocks: mocks}
	dummyMatcher := DummyMatcher{OK: true}

	r := NewRouter(dummyMapper, dummyMatcher)
	req := mock.Request{Path: "/test"}

	m, result := r.Resolve(&req)

	if len(result.Errors) > 0 || m.Response.StatusCode != 200 {
		t.Fatalf("Not route resolved")
	}

}

func TestInvalidRoute(t *testing.T) {

	mocks := []mock.Definition{
		{
			URI: "XX",
			Response: mock.Response{
				StatusCode: 200,
			},
		},
	}
	dummyMapper := DummyMapper{mocks: mocks}
	dummyMatcher := DummyMatcher{OK: false}

	r := NewRouter(dummyMapper, dummyMatcher)

	req := mock.Request{Path: "/test"}

	_, result := r.Resolve(&req)

	if len(result.Errors) == 0 || result.Errors[0].URI != "XX" || result.Errors[0].Reason != "Random Error" {
		t.Fatalf("Invalid route resolved")
	}

}

func TestResolveReturnsIndependentMock(t *testing.T) {
	mocks := []mock.Definition{
		{
			URI: "test",
			Request: mock.Request{
				HTTPEntity: mock.HTTPEntity{
					HttpHeaders: mock.HttpHeaders{
						Headers: mock.Values{"X-Test": {"request"}},
					},
				},
			},
			Response: mock.Response{
				StatusCode: 200,
				HTTPEntity: mock.HTTPEntity{
					HttpHeaders: mock.HttpHeaders{
						Headers: mock.Values{"X-Test": {"response"}},
						Cookies: mock.Cookies{"session": "original"},
					},
				},
			},
			Callback: mock.Callback{
				HTTPEntity: mock.HTTPEntity{
					HttpHeaders: mock.HttpHeaders{
						Headers: mock.Values{"X-Callback": {"callback"}},
					},
				},
			},
			Control: mock.Control{
				Scenario: mock.Scenario{
					RequiredState: []string{"started"},
				},
			},
		},
	}

	dummyMapper := DummyMapper{mocks: mocks}
	dummyMatcher := DummyMatcher{OK: true}
	r := NewRouter(dummyMapper, dummyMatcher)

	resolved, _ := r.Resolve(&mock.Request{Path: "/test"})
	resolved.Response.Headers["X-Test"][0] = "changed"
	resolved.Response.Cookies["session"] = "changed"
	resolved.Callback.Headers["X-Callback"][0] = "changed"
	resolved.Control.Scenario.RequiredState[0] = "changed"

	original := mocks[0]
	if original.Response.Headers["X-Test"][0] != "response" ||
		original.Response.Cookies["session"] != "original" ||
		original.Callback.Headers["X-Callback"][0] != "callback" ||
		original.Control.Scenario.RequiredState[0] != "started" {
		t.Fatalf("resolved mock should not share mutable data with mapping")
	}
}
