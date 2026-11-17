# List and Array Tags in Golem AIML

This document explains the `<list>` and `<array>` tags implemented in Golem, which provide powerful data structure management capabilities for AIML bots.

## Overview

Both `<list>` and `<array>` tags allow you to store and manipulate collections of data within your AIML knowledge base. They provide different semantics:

- **Lists**: Dynamic collections that grow and shrink as items are added/removed
- **Arrays**: Fixed-size collections with indexed access, automatically expanding as needed

## List Tag Syntax

### Basic Operations

```xml
<!-- Add item to list -->
<list name="listname" operation="add">value</list>

<!-- Get all items from list -->
<list name="listname"></list>

<!-- Get item at specific index -->
<list name="listname" index="0"></list>

<!-- Get list size -->
<list name="listname" operation="size"></list>
```

### Advanced Operations

```xml
<!-- Insert item at specific position -->
<list name="listname" index="2" operation="insert">value</list>

<!-- Remove item by value -->
<list name="listname" operation="remove">value</list>

<!-- Remove item at specific index -->
<list name="listname" index="1" operation="remove"></list>

<!-- Clear entire list -->
<list name="listname" operation="clear"></list>
```

## Array Tag Syntax

### Basic Operations

```xml
<!-- Set value at specific index -->
<array name="arrayname" index="0" operation="set">value</array>

<!-- Get value at specific index -->
<array name="arrayname" index="0"></array>

<!-- Get all values from array -->
<array name="arrayname"></array>

<!-- Get array size -->
<array name="arrayname" operation="size"></array>
```

### Advanced Operations

```xml
<!-- Append value to end of array -->
<array name="arrayname" operation="set">value</array>

<!-- Clear entire array -->
<array name="arrayname" operation="clear"></array>
```

## Examples

### Shopping List Management

```xml
<category>
    <pattern>ADD * TO SHOPPING LIST</pattern>
    <template>
        <list name="shopping" operation="add"><star/></list>
        I've added <star/> to your shopping list.
    </template>
</category>

<category>
    <pattern>SHOW SHOPPING LIST</pattern>
    <template>
        <list name="shopping" operation="size"></list> items in your shopping list:
        <list name="shopping"></list>
    </template>
</category>

<category>
    <pattern>REMOVE * FROM SHOPPING LIST</pattern>
    <template>
        <list name="shopping" operation="remove"><star/></list>
        I've removed <star/> from your shopping list.
    </template>
</category>
```

### Task Management with Arrays

```xml
<category>
    <pattern>SET TASK * AT POSITION *</pattern>
    <template>
        <array name="tasks" index="<star2/>" operation="set"><star/></array>
        Task "<star/>" set at position <star2/>.
    </template>
</category>

<category>
    <pattern>SHOW TASK AT POSITION *</pattern>
    <template>
        Task at position <star/>: <array name="tasks" index="<star/>"></array>
    </template>
</category>
```

### User Preferences

```xml
<category>
    <pattern>I LIKE *</pattern>
    <template>
        <list name="preferences" operation="add"><star/></list>
        I've noted that you like <star/>.
    </template>
</category>

<category>
    <pattern>WHAT DO I LIKE</pattern>
    <template>
        You like: <list name="preferences"></list>
    </template>
</category>
```

### Score Tracking

```xml
<category>
    <pattern>MY SCORE IS *</pattern>
    <template>
        <set name="current_score"><star/></set>
        <array name="scores" index="0" operation="set"><get name="current_score"/></array>
        Your score <get name="current_score"/> has been recorded.
    </template>
</category>
```

## Key Differences Between Lists and Arrays

| Feature | List | Array |
|---------|------|-------|
| **Growth** | Dynamic (add/remove items) | Fixed-size with auto-expansion |
| **Indexing** | 0-based, sequential | 0-based, can have gaps |
| **Operations** | add, insert, remove, clear | set, get, clear |
| **Use Case** | Shopping lists, preferences | Task queues, score tracking |
| **Memory** | Efficient for frequent changes | Efficient for indexed access |

## Operations Reference

### List Operations

| Operation | Description | Example |
|-----------|-------------|---------|
| `add` | Add item to end of list | `<list name="items" operation="add">apple</list>` |
| `insert` | Insert item at specific index | `<list name="items" index="2" operation="insert">banana</list>` |
| `remove` | Remove item by value or index | `<list name="items" operation="remove">apple</list>` |
| `clear` | Remove all items from list | `<list name="items" operation="clear"></list>` |
| `size` | Get number of items in list | `<list name="items" operation="size"></list>` |
| (default) | Get all items or item at index | `<list name="items" index="0"></list>` |

### Array Operations

| Operation | Description | Example |
|-----------|-------------|---------|
| `set` | Set value at specific index | `<array name="data" index="0" operation="set">value</array>` |
| `clear` | Remove all items from array | `<array name="data" operation="clear"></array>` |
| `size` | Get number of items in array | `<array name="data" operation="size"></array>` |
| (default) | Get value at index or all values | `<array name="data" index="0"></array>` |

## Error Handling

- **Invalid indices**: Non-numeric or negative indices are handled gracefully
- **Out of bounds**: Accessing non-existent indices returns empty string
- **Missing operations**: Defaults to "get" operation
- **Empty collections**: Operations on empty lists/arrays work as expected

## Best Practices

1. **Use lists for dynamic collections** that grow and shrink frequently
2. **Use arrays for indexed data** where you need specific positions
3. **Initialize collections** by adding items before accessing them
4. **Handle empty collections** gracefully in your templates
5. **Use meaningful names** for your lists and arrays
6. **Combine with variables** for dynamic content management

## Integration with Other Tags

Lists and arrays work seamlessly with other AIML tags:

- **Variables**: Use `<get>` and `<set>` to store list/array names
- **Wildcards**: Use `<star/>` to add wildcard values to collections
- **Conditions**: Check list/array contents in conditional logic
- **SRAI**: Process list/array contents through recursive calls

## Performance Considerations

- Lists are optimized for frequent additions and removals
- Arrays are optimized for indexed access
- Both collections persist across the session
- Memory usage grows with collection size
- Consider clearing unused collections periodically

## Testing

The implementation includes comprehensive tests covering:

- Basic operations (add, get, size, clear)
- Advanced operations (insert, remove)
- Error handling (invalid indices, out of bounds)
- Integration with variables and wildcards
- Persistence across multiple interactions

Run the tests with:
```bash
go test ./pkg/golem -run TestList -v
go test ./pkg/golem -run TestArray -v
```

## Demo

Try the interactive demo:
```bash
go run examples-module/list_demo.go
```

This demonstrates real-world usage patterns for both lists and arrays in a conversational AI context.
