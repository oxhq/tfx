package flowfx

import (
	"context"
	"fmt"
	"strings"
)

// Tree represents a hierarchical flow that executes steps in a tree structure.
// Each node can have children that are executed recursively.
type Tree struct {
	root       *TreeNode
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
}

// TreeNode represents a single node in the tree flow.
type TreeNode struct {
	Name     string      // Node name for identification
	Step     Step        // The step to execute at this node
	Children []*TreeNode // Child nodes to execute after this node
	Parent   *TreeNode   // Parent node (nil for root)
}

// TreeConfig provides configuration for a Tree flow.
type TreeConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
}

// DefaultTreeConfig returns the default configuration for a Tree flow.
func DefaultTreeConfig() TreeConfig {
	return TreeConfig{
		Name: "tree",
	}
}

// newTree creates a new tree flow with the given configuration.
func newTree(cfg TreeConfig) *Tree {
	return &Tree{
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewTree creates a new tree flow with multipath configuration support.
// Supports two usage patterns:
//   - NewTree()                          // Zero-config, uses defaults
//   - NewTree(config)                    // Config struct
//
// NewTree is kept as a zero-ceremony compatibility alias for MustNewTree.
// Prefer TryNewTree when args may be user-provided or otherwise fallible.
func NewTree(args ...any) *Tree {
	return MustNewTree(args...)
}

// MustNewTree creates a new tree flow and panics on invalid multipath input.
func MustNewTree(args ...any) *Tree {
	return mustValue(TryNewTree(args...))
}

// TryNewTree creates a new tree flow without panicking on invalid args.
func TryNewTree(args ...any) (*Tree, error) {
	return tryConstruct(args, DefaultTreeConfig(), newTree)
}

// NewTreeBuilder creates a new TreeBuilder for DSL chaining.
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{config: DefaultTreeConfig()}
}

// SetRoot sets the root node of the tree.
func (t *Tree) SetRoot(node *TreeNode) *Tree {
	t.root = node
	return t
}

// CreateRoot creates and sets a root node.
func (t *Tree) CreateRoot(name string, step Step) *Tree {
	t.root = &TreeNode{
		Name: name,
		Step: step,
	}
	return t
}

// Run executes the tree flow starting from the root node.
// It implements the Flow interface.
func (t *Tree) Run(ctx context.Context) error {
	if t.root == nil {
		return NewFlowError(t.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if t.onStart != nil {
		t.onStart(ctx, t.name, nil)
	}

	// Execute the tree starting from root
	if err := t.executeNode(ctx, t.root, 0); err != nil {
		if t.onError != nil {
			t.onError(ctx, t.name, err)
		}
		return err
	}

	// All nodes completed successfully
	if t.onComplete != nil {
		t.onComplete(ctx, t.name, nil)
	}

	return nil
}

// executeNode recursively executes a node and its children.
func (t *Tree) executeNode(ctx context.Context, node *TreeNode, depth int) error {
	select {
	case <-ctx.Done():
		return NewFlowError(t.name, node.Name, ErrCanceled)
	default:
	}

	// Execute the current node's step
	if node.Step != nil {
		if err := node.Step.Execute(ctx); err != nil {
			return NewFlowError(t.name, node.Name, err)
		}
	}

	// Execute all children
	for _, child := range node.Children {
		if err := t.executeNode(ctx, child, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// GetRoot returns the root node of the tree.
func (t *Tree) GetRoot() *TreeNode {
	return t.root
}

// Traverse traverses the tree and calls the visitor function for each node.
func (t *Tree) Traverse(visitor func(node *TreeNode, depth int)) {
	if t.root != nil {
		t.traverseNode(t.root, 0, visitor)
	}
}

// traverseNode recursively traverses nodes.
func (t *Tree) traverseNode(node *TreeNode, depth int, visitor func(node *TreeNode, depth int)) {
	visitor(node, depth)
	for _, child := range node.Children {
		t.traverseNode(child, depth+1, visitor)
	}
}

// String returns a string representation of the tree structure.
func (t *Tree) String() string {
	if t.root == nil {
		return "Empty tree"
	}

	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "Tree: %s\n", t.name)

	t.Traverse(func(node *TreeNode, depth int) {
		indent := strings.Repeat("  ", depth)
		_, _ = fmt.Fprintf(&sb, "%s- %s\n", indent, node.Name)
	})

	return sb.String()
}

// --- TREE NODE METHODS ---

// NewTreeNode creates a new tree node.
func NewTreeNode(name string, step Step) *TreeNode {
	return &TreeNode{
		Name:     name,
		Step:     step,
		Children: make([]*TreeNode, 0),
	}
}

// AddChild adds a child node to this node.
func (tn *TreeNode) AddChild(child *TreeNode) *TreeNode {
	child.Parent = tn
	tn.Children = append(tn.Children, child)
	return tn
}

// AddChildFunc is a convenience method to add a child with a function step.
func (tn *TreeNode) AddChildFunc(name, label string, fn func(ctx context.Context) error) *TreeNode {
	task := NewTask(label, fn)
	child := NewTreeNode(name, task)
	return tn.AddChild(child)
}

// AddChildTask is a convenience method to add a child with a task step.
func (tn *TreeNode) AddChildTask(name string, task *Task) *TreeNode {
	child := NewTreeNode(name, task)
	return tn.AddChild(child)
}

// GetChildren returns a copy of the children nodes.
func (tn *TreeNode) GetChildren() []*TreeNode {
	children := make([]*TreeNode, len(tn.Children))
	copy(children, tn.Children)
	return children
}

// IsLeaf returns true if this node has no children.
func (tn *TreeNode) IsLeaf() bool {
	return len(tn.Children) == 0
}

// IsRoot returns true if this node has no parent.
func (tn *TreeNode) IsRoot() bool {
	return tn.Parent == nil
}

// GetPath returns the path from root to this node.
func (tn *TreeNode) GetPath() []string {
	if tn.Parent == nil {
		return []string{tn.Name}
	}
	parentPath := tn.Parent.GetPath()
	return append(parentPath, tn.Name)
}

// --- DSL BUILDER ---

// TreeBuilder provides a fluent API for building tree flows.
type TreeBuilder struct {
	config TreeConfig
	root   *TreeNode
}

// Name sets the name of the tree flow.
func (tb *TreeBuilder) Name(name string) *TreeBuilder {
	tb.config.Name = name
	return tb
}

// OnStart sets the start hook.
func (tb *TreeBuilder) OnStart(hook Hook) *TreeBuilder {
	tb.config.OnStart = hook
	return tb
}

// OnComplete sets the complete hook.
func (tb *TreeBuilder) OnComplete(hook Hook) *TreeBuilder {
	tb.config.OnComplete = hook
	return tb
}

// OnError sets the error hook.
func (tb *TreeBuilder) OnError(hook Hook) *TreeBuilder {
	tb.config.OnError = hook
	return tb
}

// Root sets the root node of the tree.
func (tb *TreeBuilder) Root(node *TreeNode) *TreeBuilder {
	tb.root = node
	return tb
}

// RootFunc creates a root node with a function step.
func (tb *TreeBuilder) RootFunc(
	name, label string,
	fn func(ctx context.Context) error,
) *TreeBuilder {
	task := NewTask(label, fn)
	tb.root = NewTreeNode(name, task)
	return tb
}

// RootTask creates a root node with a task step.
func (tb *TreeBuilder) RootTask(name string, task *Task) *TreeBuilder {
	tb.root = NewTreeNode(name, task)
	return tb
}

// Build creates a new Tree instance without running it.
func (tb *TreeBuilder) Build() *Tree {
	tree := newTree(tb.config)
	tree.root = tb.root
	return tree
}

// Run creates and runs the tree flow.
func (tb *TreeBuilder) Run(ctx context.Context) error {
	return tb.Build().Run(ctx)
}

// --- CONVENIENCE BUILDERS ---

// TreeNodeBuilder provides a fluent API for building tree nodes.
type TreeNodeBuilder struct {
	node *TreeNode
}

// NewTreeNodeBuilder creates a new tree node builder.
func NewTreeNodeBuilder(name string, step Step) *TreeNodeBuilder {
	return &TreeNodeBuilder{
		node: NewTreeNode(name, step),
	}
}

// Child adds a child node.
func (tnb *TreeNodeBuilder) Child(child *TreeNode) *TreeNodeBuilder {
	tnb.node.AddChild(child)
	return tnb
}

// ChildFunc adds a child with a function step.
func (tnb *TreeNodeBuilder) ChildFunc(
	name, label string,
	fn func(ctx context.Context) error,
) *TreeNodeBuilder {
	tnb.node.AddChildFunc(name, label, fn)
	return tnb
}

// ChildTask adds a child with a task step.
func (tnb *TreeNodeBuilder) ChildTask(name string, task *Task) *TreeNodeBuilder {
	tnb.node.AddChildTask(name, task)
	return tnb
}

// Build returns the built tree node.
func (tnb *TreeNodeBuilder) Build() *TreeNode {
	return tnb.node
}
