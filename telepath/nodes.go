package telepath

import (
	"slices"
)

var _ Node = (*TelepathNode)(nil)

type TelepathNode struct {
	ID            int
	Seen          bool
	UseIdentifier bool
	EmitVerboseFn func() TelepathValue
	EmitCompactFn func() any
}

func NewTelepathNode() *TelepathNode {
	return &TelepathNode{}
}

func (m *TelepathNode) GetValue() interface{} {
	return nil
}

func (m *TelepathNode) SetID(id int) {
	m.ID = id
}

func (m *TelepathNode) GetID() int {
	return m.ID
}

func (m *TelepathNode) UseID() bool {
	return m.UseIdentifier
}

func (m *TelepathNode) Emit() any {
	if m.UseID() {
		return TelepathValue{Ref: m.ID}
	}

	m.Seen = true
	if m.UseID() && m.ID != 0 {
		var result = m.EmitVerbose()
		result.ID = m.ID
		return result
	}

	return m.EmitCompact()
}

func (m *TelepathNode) EmitVerbose() TelepathValue {
	return TelepathValue{}
}

func (m *TelepathNode) EmitCompact() any {
	return nil
}

type TelepathValueNode struct {
	*TelepathNode
	Value interface{}
}

func NewTelepathValueNode(value interface{}) *TelepathValueNode {
	return &TelepathValueNode{
		Value:        value,
		TelepathNode: NewTelepathNode(),
	}
}

func (m *TelepathValueNode) UseID() bool {
	return false
}

func (m *TelepathValueNode) GetValue() interface{} {
	return m.Value
}

func (m *TelepathValueNode) EmitVerbose() TelepathValue {
	return TelepathValue{Val: m.GetValue()}
}

func (m *TelepathValueNode) EmitCompact() any {
	return m.GetValue()
}

type StringNode struct {
	*TelepathValueNode
}

func (m *StringNode) UseID() bool {
	return len(m.GetValue().(string)) >= STRING_REF_MIN_LENGTH
}

func NewStringNode(value interface{}) *StringNode {
	return &StringNode{
		TelepathValueNode: NewTelepathValueNode(value),
	}
}

func (m *StringNode) Emit() any {
	if m.UseID() {
		return TelepathValue{Ref: m.ID}
	}

	m.Seen = true
	if m.UseID() && m.ID != 0 {
		var result = m.EmitVerbose()
		result.ID = m.ID
		return result
	}

	return m.EmitCompact()
}

func (m *StringNode) EmitVerbose() TelepathValue {
	return TelepathValue{Val: m.GetValue()}
}

func (m *StringNode) EmitCompact() any {
	return m.GetValue()
}

type ObjectNode struct {
	*TelepathValueNode
	Constructor string
	Args        []Node
}

func NewObjectNode(constructor string, args []Node) *ObjectNode {
	return &ObjectNode{
		TelepathValueNode: NewTelepathValueNode(nil),
		Constructor:       constructor,
		Args:              args,
	}
}

func (m *ObjectNode) Emit() any {
	return m.EmitVerbose()
}

func (m *ObjectNode) EmitVerbose() TelepathValue {
	var result = TelepathValue{
		Type: m.Constructor,
		Args: make([]any, 0, len(m.Args)),
	}
	for _, arg := range m.Args {
		result.Args = append(result.Args, arg.Emit())
	}
	return result
}

func (m *ObjectNode) EmitCompact() any {
	return m.EmitVerbose()
}

type DictNode struct {
	*TelepathValueNode
}

func NewDictNode(value map[string]Node) *DictNode {
	return &DictNode{
		TelepathValueNode: NewTelepathValueNode(value),
	}
}

func (m *DictNode) Emit() any {
	if m.UseID() {
		return TelepathValue{Ref: m.ID}
	}

	m.Seen = true
	if m.UseID() && m.ID != 0 {
		var result = m.EmitVerbose()
		result.ID = m.ID
		return result
	}

	return m.EmitCompact()
}

func (m *DictNode) EmitVerbose() TelepathValue {
	var result = TelepathValue{Dict: make(map[string]interface{})}
	for key, value := range m.Value.(map[string]Node) {
		result.Dict[key] = value.Emit()
	}
	return result
}

func (m *DictNode) EmitCompact() any {
	var (
		hasReservedKey = false
		result         = make(map[string]interface{})
	)

	for key := range m.Value.(map[string]Node) {
		_, hasReservedKey = slices.BinarySearch(
			DICT_RESERVED_KEYS, key,
		)
		if hasReservedKey {
			return m.EmitVerbose()
		}
	}

	for key, value := range m.Value.(map[string]Node) {
		result[key] = value.Emit()
	}

	return result
}

type ListNode struct {
	*TelepathValueNode
}

func NewListNode(value []Node) *ListNode {
	return &ListNode{
		TelepathValueNode: NewTelepathValueNode(value),
	}
}

func (m *ListNode) GetValue() interface{} {
	return m.Value
}

func (m *ListNode) Emit() any {
	return m.EmitVerbose()
}

func (m *ListNode) EmitVerbose() TelepathValue {
	var result = TelepathValue{List: make([]interface{}, 0)}
	for _, value := range m.Value.([]Node) {
		result.List = append(result.List, value.Emit())
	}
	return result
}

func (m *ListNode) EmitCompact() any {
	var result = make([]interface{}, 0)
	for _, value := range m.Value.([]Node) {
		result = append(result, value.Emit())
	}
	return result
}
