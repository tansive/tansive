package tangentcommon

import (
	"testing"
)

func TestStack_PushPop(t *testing.T) {
	stack := NewStack[int]()

	stack.Push(10)
	stack.Push(20)

	if stack.Len() != 2 {
		t.Errorf("expected length 2, got %d", stack.Len())
	}

	val, ok := stack.Pop()
	if !ok || val != 20 {
		t.Errorf("expected Pop to return 20, got %v (ok=%v)", val, ok)
	}

	val, ok = stack.Pop()
	if !ok || val != 10 {
		t.Errorf("expected Pop to return 10, got %v (ok=%v)", val, ok)
	}

	if !stack.IsEmpty() {
		t.Error("expected stack to be empty after popping all items")
	}

	_, ok = stack.Pop()
	if ok {
		t.Error("expected Pop on empty stack to return ok=false")
	}
}

func TestStack_Peek(t *testing.T) {
	stack := NewStack[string]()

	_, ok := stack.Peek()
	if ok {
		t.Error("expected Peek on empty stack to return ok=false")
	}

	stack.Push("foo")
	stack.Push("bar")

	val, ok := stack.Peek()
	if !ok || val != "bar" {
		t.Errorf("expected Peek to return 'bar', got %v (ok=%v)", val, ok)
	}

	if stack.Len() != 2 {
		t.Errorf("expected length to remain 2 after Peek, got %d", stack.Len())
	}
}
