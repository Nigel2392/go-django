package todos

import "context"

func ListAllTodos(ctx context.Context, limit, offset int) ([]Todo, error) {
	var rows, err = globalDB.QueryContext(ctx, listTodos, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Done); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func GetTodoByID(ctx context.Context, id int) (*Todo, error) {
	var todo Todo
	if err := globalDB.QueryRowContext(ctx, selectTodo, id).Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Done); err != nil {
		return nil, err
	}
	return &todo, nil
}

func CountTodos(ctx context.Context) (int, error) {
	var count int
	if err := globalDB.QueryRowContext(ctx, countTodos).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
