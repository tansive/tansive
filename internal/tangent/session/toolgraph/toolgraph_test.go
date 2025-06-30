package toolgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterCall_BasicChain(t *testing.T) {
	g := NewCallGraph(0)

	err := g.RegisterCall("", "ToolA", "a1")
	assert.NoError(t, err)

	err = g.RegisterCall("a1", "ToolB", "b1")
	assert.NoError(t, err)

	err = g.RegisterCall("b1", "ToolC", "c1")
	assert.NoError(t, err)

	lineage := g.DebugGraph("c1")
	assert.Equal(t, []string{
		"a1 (ToolA)",
		"b1 (ToolB)",
		"c1 (ToolC)",
	}, lineage)
}

func TestRegisterCall_LoopDetection(t *testing.T) {
	g := NewCallGraph(0)

	_ = g.RegisterCall("", "ToolA", "a1")
	_ = g.RegisterCall("a1", "ToolB", "b1")
	_ = g.RegisterCall("b1", "ToolC", "c1")

	// Now attempt to call ToolA again under ToolC — should detect loop
	err := g.RegisterCall("c1", "ToolA", "a2")
	assert.ErrorContains(t, err, "loop detected")
}

func TestRegisterCall_DepthLimit(t *testing.T) {
	g := NewCallGraph(4)

	_ = g.RegisterCall("", "ToolA", "a1")
	_ = g.RegisterCall("a1", "ToolB", "b1")
	_ = g.RegisterCall("b1", "ToolC", "c1")

	// Should succeed with limit 4
	err := g.RegisterCall("c1", "ToolD", "d1")
	assert.NoError(t, err)

	g = NewCallGraph(3)
	// Should fail with limit 3 (a1 → b1 → c1 already = depth 3)
	_ = g.RegisterCall("", "ToolA", "a1")
	_ = g.RegisterCall("a1", "ToolB", "b1")
	_ = g.RegisterCall("b1", "ToolC", "c1")

	err = g.RegisterCall("c1", "ToolE", "e1")
	assert.ErrorContains(t, err, "call depth limit exceeded")
}

func TestRegisterCall_NoLimitDepthZero(t *testing.T) {
	g := NewCallGraph(0)

	err := g.RegisterCall("", "ToolA", "a1")
	assert.NoError(t, err)

	err = g.RegisterCall("a1", "ToolB", "b1")
	assert.NoError(t, err)

	err = g.RegisterCall("b1", "ToolC", "c1")
	assert.NoError(t, err)

	err = g.RegisterCall("c1", "ToolD", "d1")
	assert.NoError(t, err)

	assert.Equal(t, []string{
		"a1 (ToolA)",
		"b1 (ToolB)",
		"c1 (ToolC)",
		"d1 (ToolD)",
	}, g.DebugGraph("d1"))
}

func TestRegisterCall_ParallelBranches(t *testing.T) {
	g := NewCallGraph(0)

	_ = g.RegisterCall("", "ToolA", "a1")

	// A → B and A → C (parallel branches)
	err := g.RegisterCall("a1", "ToolB", "b1")
	assert.NoError(t, err)

	err = g.RegisterCall("a1", "ToolC", "c1")
	assert.NoError(t, err)

	// B calls D
	err = g.RegisterCall("b1", "ToolD", "d1")
	assert.NoError(t, err)

	// C calls D — still OK since no loop on tool name in this path
	err = g.RegisterCall("c1", "ToolD", "d2")
	assert.NoError(t, err)

	// D2 tries to call A — should detect loop via A
	err = g.RegisterCall("d2", "ToolA", "a2")
	assert.ErrorContains(t, err, "loop detected")
}
