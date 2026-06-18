//go:build js && wasm

package provider

import (
	"context"
	"testing"
)

// alwaysProvider satisfies Provider and is always available.
type alwaysProvider struct{ name string }

func (p *alwaysProvider) Name() string                                          { return p.name }
func (p *alwaysProvider) MaxContextWindow(_ string) int                         { return 4096 }
func (p *alwaysProvider) Complete(_ context.Context, _ CompletionRequest) (*Token, error) {
	return &Token{Text: "ok", FinishReason: "stop"}, nil
}
func (p *alwaysProvider) Stream(_ context.Context, _ CompletionRequest) <-chan Token {
	ch := make(chan Token, 1)
	ch <- Token{Text: "ok", FinishReason: "stop"}
	close(ch)
	return ch
}
func (p *alwaysProvider) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }

// conditionalProvider extends alwaysProvider with a runtime gate.
type conditionalProvider struct {
	alwaysProvider
	available bool
}

func (p *conditionalProvider) IsAvailable() bool { return p.available }

func TestAvailableProviders_ConditionalFiltering(t *testing.T) {
	r := &Router{
		providers: make(map[string]*ProviderChain),
		fallbacks: make(map[string]fallbackConfig),
	}

	r.providers["always"] = NewProviderChain(&alwaysProvider{name: "always"}, "model-a")
	r.providers["present"] = NewProviderChain(&conditionalProvider{alwaysProvider{"present"}, true}, "model-b")
	r.providers["absent"] = NewProviderChain(&conditionalProvider{alwaysProvider{"absent"}, false}, "model-c")

	got := r.AvailableProviders()

	has := func(name string) bool {
		for _, n := range got {
			if n == name {
				return true
			}
		}
		return false
	}

	if !has("always") {
		t.Error("unconditional provider 'always' should be included")
	}
	if !has("present") {
		t.Error("conditional provider with IsAvailable()==true should be included")
	}
	if has("absent") {
		t.Error("conditional provider with IsAvailable()==false should be excluded")
	}
}

func TestAvailableProviders_Empty(t *testing.T) {
	r := &Router{
		providers: make(map[string]*ProviderChain),
		fallbacks: make(map[string]fallbackConfig),
	}
	if got := r.AvailableProviders(); len(got) != 0 {
		t.Errorf("want empty slice, got %v", got)
	}
}
